//   Copyright 2017 MSolution.IO
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package costs

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gopkg.in/olivere/elastic.v5"

	"github.com/trackit/jsonlog"
	"github.com/trackit/trackit-server/aws/s3"
	"github.com/trackit/trackit-server/db"
	"github.com/trackit/trackit-server/errors"
	"github.com/trackit/trackit-server/es"
	"github.com/trackit/trackit-server/routes"
	"github.com/trackit/trackit-server/users"
)

// esQueryParams will store the parsed query params
type esQueryParams struct {
	dateBegin   time.Time
	dateEnd     time.Time
	accountList []string
	indexList   []string
}

// esFilter represents an elasticsearch filter
type esFilter struct {
	Key   string
	Value string
}

type esFilters = []esFilter

// queryDataTypeToEsFilters represents the different types of data
// that can be requested from ES and their associated slice of filters
var queryDataTypeToEsFilters = map[string]esFilters{
	"storage": esFilters{
		esFilter{"usageType", "*TimedStorage*"},
	},
	"requests": esFilters{
		esFilter{"usageType", "*Requests*"},
	},
	"bandwidthIn": esFilters{
		esFilter{"usageType", "*In*"},
		esFilter{"serviceCode", "AWSDataTransfer"},
	},
	"bandwidthOut": esFilters{
		esFilter{"usageType", "*Out*"},
		esFilter{"serviceCode", "AWSDataTransfer"},
	},
}

func init() {
	routes.MethodMuxer{
		http.MethodGet: routes.H(getS3CostData).With(
			db.RequestTransaction{Db: db.Db},
			users.RequireAuthenticatedUser{users.ViewerAsParent},
			routes.Documentation{
				Summary:     "get the s3 costs data",
				Description: "Responds with cost data based on the queryparams passed to it",
			},
			routes.QueryArgs{routes.AwsAccountsOptionalQueryArg},
			routes.QueryArgs{routes.DateBeginQueryArg},
			routes.QueryArgs{routes.DateEndQueryArg},
		),
	}.H().Register("/s3/costs")
}

// makeElasticSearchRequest prepares and run the request to retrieve usage and cost
// informations related to the queryDataType
// It will return the data, an http status code (as int) and an error.
// Because an error can be generated, but is not critical and is not needed to be known by
// the user (e.g if the index does not exists because it was not yet indexed ) the error will
// be returned, but instead of having a 500 status code, it will return the provided status code
// with empy data
func makeElasticSearchRequest(ctx context.Context, parsedParams esQueryParams,
	queryDataType string) (*elastic.SearchResult, int, error) {
	l := jsonlog.LoggerFromContextOrDefault(ctx)
	index := strings.Join(parsedParams.indexList, ",")

	esFilters, ok := queryDataTypeToEsFilters[queryDataType]
	if ok == false {
		err := fmt.Errorf("QueryDataType '%s' not found", queryDataType)
		l.Error("Failed to retrieve s3 costs", err)
		return nil, http.StatusInternalServerError, err
	}

	searchService := GetS3UsageAndCostElasticSearchParams(
		parsedParams.accountList,
		parsedParams.dateBegin,
		parsedParams.dateEnd,
		esFilters,
		es.Client,
		index,
	)
	res, err := searchService.Do(ctx)
	if err != nil {
		if elastic.IsNotFound(err) {
			l.Warning("Query execution failed, ES index does not exists : "+index, err)
			return nil, http.StatusOK, errors.GetErrorMessage(ctx, err)
		}
		l.Error("Query execution failed : "+err.Error(), nil)
		return nil, http.StatusInternalServerError, fmt.Errorf("could not execute the ElasticSearch query")
	}
	return res, http.StatusOK, nil
}

// getS3CostData returns the s3 cost data based on the query params, in JSON format.
func getS3CostData(request *http.Request, a routes.Arguments) (int, interface{}) {
	user := a[users.AuthenticatedUser].(users.User)
	parsedParams := esQueryParams{
		dateBegin:   a[routes.DateBeginQueryArg].(time.Time),
		dateEnd:     a[routes.DateEndQueryArg].(time.Time).Add(time.Hour*time.Duration(23) + time.Minute*time.Duration(59) + time.Second*time.Duration(59)),
		accountList: []string{},
	}
	if a[routes.AwsAccountsOptionalQueryArg] != nil {
		parsedParams.accountList = a[routes.AwsAccountsOptionalQueryArg].([]string)
	}
	var err error
	var returnCode int
	tx := a[db.Transaction].(*sql.Tx)
	accountsAndIndexes, returnCode, err := es.GetAccountsAndIndexes(parsedParams.accountList, user, tx, s3.IndexPrefixLineItem)
	if err != nil {
		return returnCode, err
	}
	parsedParams.accountList = accountsAndIndexes.Accounts
	parsedParams.indexList = accountsAndIndexes.Indexes
	var components = [...]struct {
		k  string
		sr *elastic.SearchResult
	}{
		{"storage", nil},
		{"requests", nil},
		{"bandwidthIn", nil},
		{"bandwidthOut", nil},
	}
	for idx, cpn := range components {
		cpn.sr, returnCode, err = makeElasticSearchRequest(request.Context(), parsedParams, cpn.k)
		if err != nil {
			return returnCode, err
		}
		components[idx] = cpn
	}
	res, err := prepareResponse(
		request.Context(),
		components[0].sr,
		components[1].sr,
		components[2].sr,
		components[3].sr,
	)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, res
}

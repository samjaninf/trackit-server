//   Copyright 2018 MSolution.IO
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

package reservedInstances

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/trackit/jsonlog"

	taws "github.com/trackit/trackit-server/aws"
	"github.com/trackit/trackit-server/aws/usageReports"
	"github.com/trackit/trackit-server/config"
)

// fetchDailyInstancesList sends in instanceInfoChan the instances fetched from DescribeInstances
// and filled by DescribeInstances and getInstanceStats.
func fetchDailyInstancesList(ctx context.Context, creds *credentials.Credentials, region string, instanceChan chan Instance) error {
	defer close(instanceChan)
	//start, end := utils.GetCurrentCheckedDay()
	logger := jsonlog.LoggerFromContextOrDefault(ctx)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(region),
	}))
	svc := ec2.New(sess)
	reservedInstances, err := svc.DescribeReservedInstances(nil)
	if err != nil {
		logger.Error("Error when describing instances", err.Error())
		return err
	}
	for _, reservation := range reservedInstances.ReservedInstances {
		logger.Debug("RI : ", reservation)
		//stats := getInstanceStats(ctx, reservation, sess, start, end)
		instanceChan <- Instance{
			Id:              aws.StringValue(reservation.ReservedInstancesId),
			Region:          aws.StringValue(reservation.AvailabilityZone),
			Tags:            getInstanceTag(reservation.Tags),
			Type:            aws.StringValue(reservation.InstanceType),
			FixedPrice:      aws.Float64Value(reservation.FixedPrice),
			UsagePrice:      aws.Float64Value(reservation.UsagePrice),
			Duration:        aws.Int64Value(reservation.Duration),
			Start:           aws.TimeValue(reservation.Start),
			End:             aws.TimeValue(reservation.End),
			InstanceCount:   aws.Int64Value(reservation.InstanceCount),
			InstanceTenancy: aws.StringValue(reservation.InstanceTenancy),
			//OfferingClass: aws.StringValue(reservation.OfferingClass),
			//OfferingType: aws.StringValue(reservation.OfferingType),
			//ProductDescription: aws.StringValue(reservation.ProductDescription),
		}
	}
	return nil
}

// FetchDailyInstancesStats fetches the stats of the EC2 instances of an AwsAccount
// to import them in ElasticSearch. The stats are fetched from the last hour.
// In this way, FetchInstancesStats should be called every hour.
func FetchDailyInstancesStats(ctx context.Context, awsAccount taws.AwsAccount) error {
	logger := jsonlog.LoggerFromContextOrDefault(ctx)
	logger.Info("Fetching EC2 instance stats", map[string]interface{}{"awsAccountId": awsAccount.Id})
	creds, err := taws.GetTemporaryCredentials(awsAccount, MonitorInstanceStsSessionName)
	if err != nil {
		logger.Error("Error when getting temporary credentials", err.Error())
		return err
	}
	defaultSession := session.Must(session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(config.AwsRegion),
	}))
	now := time.Now().UTC()
	account, err := utils.GetAccountId(ctx, defaultSession)
	if err != nil {
		logger.Error("Error when getting account id", err.Error())
		return err
	}
	regions, err := utils.FetchRegionsList(ctx, defaultSession)
	if err != nil {
		logger.Error("Error when fetching regions list", err.Error())
		return err
	}
	instanceChans := make([]<-chan Instance, 0, len(regions))
	for _, region := range regions {
		instanceChan := make(chan Instance)
		go fetchDailyInstancesList(ctx, creds, region, instanceChan)
		instanceChans = append(instanceChans, instanceChan)
	}
	instances := make([]InstanceReport, 0)
	for instance := range merge(instanceChans...) {
		instances = append(instances, InstanceReport{
			Account:    account,
			ReportDate: now,
			ReportType: "daily",
			Instance:   instance,
		})
	}
	return importInstancesToEs(ctx, awsAccount, instances)
}
// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"tavern.aws/org-tool/pkg/models"
)

var (
	stsc    *sts.Client
	orgc    *organizations.Client
	ec2c    *ec2.Client
	regions []string
)

// init initializes common AWS SDK clients and pulls in all enabled regions
func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile("tavern-automation"))
	if err != nil {
		log.Fatal("ERROR: Unable to resolve credentials for tavern-automation: ", err)
	}

	stsc = sts.NewFromConfig(cfg)
	orgc = organizations.NewFromConfig(cfg)
	ec2c = ec2.NewFromConfig(cfg)

	// NOTE: By default, only describes regions that are enabled in the root org account, not all Regions
	resp, err := ec2c.DescribeRegions(context.TODO(), &ec2.DescribeRegionsInput{})
	if err != nil {
		log.Fatal("ERROR: Unable to describe regions", err)
	}

	for _, region := range resp.Regions {
		regions = append(regions, *region.RegionName)
	}
	fmt.Println("INFO: Listing all enabled regions:")
	fmt.Println(regions)
}

// main constructs a concurrent pipeline that pushes every account ID down
// the pipeline, where an action is concurrently run on each account and
// results are aggregated into a single json file
func main() {
	var accounts []string

	paginator := organizations.NewListAccountsPaginator(orgc, &organizations.ListAccountsInput{})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Fatal("ERROR: Unable to list accounts in this organization: ", err)
		}

		for _, account := range resp.Accounts {
			accounts = append(accounts, *account.Id)
		}
	}
	fmt.Println(accounts)

	// Begin pipeline by calling gen with a list of every account
	in := gen(accounts...)

	// Fan out and create individual goroutines handling the requested action (getRoute)
	var out []<-chan models.InternetRoute
	for range accounts {
		c := getRoute(in)
		out = append(out, c)
	}

	// Fans in and collect the routing information from all go routines
	var allRoutes []models.InternetRoute
	for n := range merge(out...) {
		allRoutes = append(allRoutes, n)
	}

	savedRoutes, err := json.MarshalIndent(allRoutes, "", "\t")
	if err != nil {
		fmt.Println("ERROR: Unable to marshal internet routes to JSON: ", err)
	}
	ioutil.WriteFile("routes.json", savedRoutes, 0644)
}

// gen primes the pipeline, creating a single separate goroutine
// that will sequentially put a single account id down the channel
// gen returns the channel so that we can plug it in into the next
// stage
func gen(accounts ...string) <-chan string {
	out := make(chan string)
	go func() {
		for _, account := range accounts {
			out <- account
		}
		close(out)
	}()
	return out
}

// getRoute queries every route table in an account, including every enabled region, for a
// 0.0.0.0/0 (i.e. default route) to an internet gateway
func getRoute(in <-chan string) <-chan models.InternetRoute {
	out := make(chan models.InternetRoute)
	go func() {
		for account := range in {
			role := fmt.Sprintf("arn:aws:iam::%s:role/TavernAutomationRole", account)
			creds := stscreds.NewAssumeRoleProvider(stsc, role)

			for _, region := range regions {
				localCfg := aws.Config{
					Region:      region,
					Credentials: aws.NewCredentialsCache(creds),
				}

				localEc2Client := ec2.NewFromConfig(localCfg)

				paginator := ec2.NewDescribeRouteTablesPaginator(localEc2Client, &ec2.DescribeRouteTablesInput{})
				for paginator.HasMorePages() {
					resp, err := paginator.NextPage(context.TODO())
					if err != nil {
						fmt.Println("WARNING: Unable to retrieve route tables from account: ", account, err)
						out <- models.InternetRoute{Account: account}
						close(out)
						return
					}

					for _, routeTable := range resp.RouteTables {
						for _, r := range routeTable.Routes {
							if r.GatewayId != nil && strings.Contains(*r.GatewayId, "igw-") {
								fmt.Println(
									"Account: ", account,
									" Region: ", region,
									" DestinationCIDR: ", *r.DestinationCidrBlock,
									" GatewayId: ", *r.GatewayId,
								)
	
								out <- models.InternetRoute{
									Account:         account,
									Region:          region,
									Vpc:             routeTable.VpcId,
									RouteTable:      routeTable.RouteTableId,
									DestinationCidr: r.DestinationCidrBlock,
									InternetGateway: r.GatewayId,
								}
							}
						}
					}
				}
			}

		}
		close(out)
	}()
	return out
}

// merge takes every go routine and "plugs" it into a common out channel
// then blocks until every input channel closes, signally that all goroutines
// are done in the previous stage
func merge(cs ...<-chan models.InternetRoute) <-chan models.InternetRoute {
	var wg sync.WaitGroup
	out := make(chan models.InternetRoute)

	output := func(c <-chan models.InternetRoute) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}

	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

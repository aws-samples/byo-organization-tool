// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

package models

type InternetRoute struct {
	Account         string
	Region          string
	Vpc             *string
	RouteTable      *string
	DestinationCidr *string
	InternetGateway *string
}

// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

import * as cdk from '@aws-cdk/core';
import * as iam from '@aws-cdk/aws-iam';
import { PolicyStatement } from '@aws-cdk/aws-iam';

export class StacksetStack extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // cdk synth -c OrgAccount={OrgAccount} -c RoleName={RoleName} -c SSOUser={SSOUser} > tavernauto.yaml
    const orgAccount = this.node.tryGetContext('OrgAccount');
    const roleName = this.node.tryGetContext('RoleName');
    const ssoUser = this.node.tryGetContext('SSOUser');

    // aws cloudformation update-stack-set --stack-set-name TavernAutomations --template-body file://tavernauto.yaml --deployment-targets OrganizationalUnitIds='[r-*]' --profile ProfileName --regions us-east-1 --operation-preferences MaxConcurrentPercentage=100,FailureToleranceCount=5 --capabilities CAPABILITY_NAMED_IAM 
    const role = new iam.Role(this, 'TavernAutomationRole', {
      roleName: 'TavernAutomationRole',
      assumedBy: new iam.ArnPrincipal(`arn:aws:sts::${orgAccount}:assumed-role/${roleName}/${ssoUser}`),
    })
    role.addToPolicy(new PolicyStatement({
      actions: ['ec2:DescribeRouteTables'],
      resources: ['*']
    }))

  }
}

# Organization Tool

This is sample code from the blog "Building your own organization supported tools"

## Requirements

Both go and nodejs need to be installed locally

## Getting Started

1. Clone this repository locally
2. Install all go dependencies with `go mod tidy`
3. `cd stackset` to change directory into stackset
4. Run `npm install` to install all CDK dependencies
5. Ready to go!

## Commands

### Deploying the Common Role Stack

To synthesize the stack, run:

- `cdk synth -c OrgAccount={OrgAccount} -c RoleName={RoleName} -c SSOUser={SSOUser} > tavernauto.yaml`

This will synthesize the CDK into cloudformation which can be deployed via Stack Sets in the AWS Console

### Running the tool

Ensure your authenticated with `aws sso login --profile OrgAccountProfile` and run `go run main.go`

## Security

See [CONTRIBUTING](CONTRIBUTING.md#security-issue-notifications) for more information.

## License

This library is licensed under the MIT-0 License. See the LICENSE file.


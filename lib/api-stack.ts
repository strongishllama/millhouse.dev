import * as cdk from '@aws-cdk/core';
import * as apigateway from '@aws-cdk/aws-apigateway';
import * as certificatemanager from '@aws-cdk/aws-certificatemanager';
import * as dynamodb from '@aws-cdk/aws-dynamodb';
import * as iam from '@aws-cdk/aws-iam';
import * as go_lambda from '@aws-cdk/aws-lambda-go';
import * as route53 from '@aws-cdk/aws-route53';
import * as targets from '@aws-cdk/aws-route53-targets';
import * as ssm from '@aws-cdk/aws-ssm';
import * as sqs from '@aws-cdk/aws-sqs';
import { DynamoDB, SecretsManager, SQS } from '@strongishllama/iam-constants-cdk';
import { bundling } from './lambda';
import { Stage } from './stage';
import { Method } from './method';

export interface ApiStackProps extends cdk.StackProps {
  namespace: string;
  stage: Stage;
  lambdasConfigArn: string;
  adminTo: string;
  adminFrom: string;
}

export class ApiStack extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props: ApiStackProps) {
    super(scope, id, props);

    // Fetch the table via the table ARN.
    const table = dynamodb.Table.fromTableArn(
      this,
      `${props.namespace}-subscription-table${props.stage}`,
      ssm.StringParameter.fromStringParameterName(this, `${props.namespace}-subscription-table-arn-${props.stage}`, `${props.namespace}-table-arn-${props.stage}`).stringValue
    );

    // Fetch the email queue via the queue ARN.
    const emailQueue = sqs.Queue.fromQueueArn(
      this,
      `${props.namespace}-email-queue-${props.stage}`,
      ssm.StringParameter.fromStringParameterName(this, `${props.namespace}-email-queue-arn-${props.stage}`, `${props.namespace}-email-queue-${props.stage}`).stringValue
    );

    // Create a REST API for the website to interact with.
    const api = new apigateway.RestApi(this, `${props.namespace}-rest-api-${props.stage}`, {
      defaultCorsPreflightOptions: {
        allowOrigins: props.stage === Stage.PROD ? ["https://millhouse.dev"] : apigateway.Cors.ALL_ORIGINS
      },
      deployOptions: {
        stageName: props.stage
      },
    });

    // Add ping method - /
    api.root.addMethod(Method.GET, new apigateway.LambdaIntegration(new go_lambda.GoFunction(this, `${props.namespace}-ping-function-${props.stage}`, {
      entry: 'lambdas/api/ping',
      bundling: bundling,
      environment: {
        'STAGE': props.stage
      }
    })));

    // Add subscribe method - /subscribe
    api.root.addResource('subscribe').addMethod(Method.PUT, new apigateway.LambdaIntegration(new go_lambda.GoFunction(this, `${props.namespace}-subscribe-function-${props.stage}`, {
      entry: 'lambdas/api/subscribe',
      bundling: bundling,
      environment: {
        'ADMIN_TO': props.adminTo,
        'ADMIN_FROM': props.adminFrom,
        'CONFIG_SECRET_ARN': props.lambdasConfigArn,
        'EMAIL_QUEUE_URL': emailQueue.queueUrl,
        'STAGE': props.stage,
        'TABLE_NAME': table.tableName
      },
      initialPolicy: [
        new iam.PolicyStatement({
          actions: [
            SecretsManager.GET_SECRET_VALUE
          ],
          resources: [
            props.lambdasConfigArn
          ]
        }),
        new iam.PolicyStatement({
          actions: [
            DynamoDB.PUT_ITEM,
            DynamoDB.QUERY,
          ],
          resources: [
            table.tableArn,
            `${table.tableArn}/index/*`
          ]
        }),
        new iam.PolicyStatement({
          actions: [
            SQS.SEND_MESSAGE
          ],
          resources: [
            emailQueue.queueArn
          ]
        }),
      ]
    })));

    // Add unsubscribe method - /unsubscribe
    api.root.addResource('unsubscribe').addMethod(Method.GET, new apigateway.LambdaIntegration(new go_lambda.GoFunction(this, `${props.namespace}-unsubscribe-function-${props.stage}`, {
      entry: 'lambdas/api/unsubscribe',
      bundling: bundling,
      environment: {
        'STAGE': props.stage,
        'TABLE_NAME': table.tableName
      },
      initialPolicy: [
        new iam.PolicyStatement({
          actions: [
            DynamoDB.DELETE_ITEM
          ],
          resources: [
            table.tableArn
          ]
        })
      ]
    })));

    // Fetch hosted zone via the domain name.
    const hostedZone = route53.HostedZone.fromLookup(this, `${props.namespace}-hosted-zone-${props.stage}`, {
      domainName: 'millhouse.dev'
    });

    // Determine the full domain name based on the stage.
    const fullDomainName = props.stage === Stage.PROD ? 'api.millhouse.dev' : `${props.stage}.api.millhouse.dev`;

    // Create a DNS validated certificate for HTTPS
    const certificate = new certificatemanager.DnsValidatedCertificate(this, `${props.namespace}-api-certificate-${props.stage}`, {
      domainName: fullDomainName,
      hostedZone: route53.HostedZone.fromLookup(this, `${props.namespace}-api-hosted-zone-${props.stage}`, {
        domainName: 'millhouse.dev'
      })
    });

    // Create a domain name for the API and map it.
    const domain = new apigateway.DomainName(this, `${props.namespace}-api-domain-name-${props.stage}`, {
      domainName: fullDomainName,
      certificate: certificate,
    });
    domain.addBasePathMapping(api);

    // Create an A record pointing at the web distribution.
    new route53.ARecord(this, `${props.namespace}-a-record-${props.stage}`, {
      zone: hostedZone,
      recordName: fullDomainName,
      ttl: cdk.Duration.seconds(60),
      target: route53.RecordTarget.fromAlias(new targets.ApiGatewayDomain(domain))
    });
  }

}
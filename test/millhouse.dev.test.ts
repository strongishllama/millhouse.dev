import { expect as expectCDK, matchTemplate, MatchStyle } from '@aws-cdk/assert';
import * as cdk from '@aws-cdk/core';
import * as MillhouseDev from '../lib/millhouse.dev-stack';

test('Empty Stack', () => {
    const app = new cdk.App();
    // WHEN
    const stack = new MillhouseDev.MillhouseDevStack(app, 'MyTestStack');
    // THEN
    expectCDK(stack).to(matchTemplate({
      "Resources": {}
    }, MatchStyle.EXACT))
});

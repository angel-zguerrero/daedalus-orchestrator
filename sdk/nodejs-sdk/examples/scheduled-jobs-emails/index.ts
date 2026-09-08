import { DaedalusSDK, ClaimedMessage, AckCallback } from '../../src/index';

interface ScheduledEmail {
  subject: string;
  recipient: string;
  body: string;
  type: string;
}

async function main() {
  const sdk = new DaedalusSDK({
    uri: 'http://localhost:4000',
    username: 'admin',
    password: 'admin',
  });

  try {
    await sdk.connect();

    const tenantCode = 'acme-corp';
    const vnamespace = 'default';

    // 1. Assert Tenant
    await sdk.assertTenant({
      code: tenantCode,
      name: 'Acme Corporation',
    });

    // 2. Assert Exchange
    await sdk.assertExchange({
      tenantCode: tenantCode,
      code: 'email-events',
      name: 'Email Events Exchange',
      type: 'topic',
      vnamespace: vnamespace,
    });

    // 3. Assert Queues
    await sdk.assertQueue({
      tenantCode: tenantCode,
      code: 'email-delayed',
      name: 'Delayed Email Queue',
      type: 'standard',
      state: 'active',
      vnamespace: vnamespace,
      maxAttempts: 3,
    });

    await sdk.assertQueue({
      tenantCode: tenantCode,
      code: 'weekly-reports',
      name: 'Weekly Reports Queue',
      type: 'standard',
      state: 'active',
      vnamespace: vnamespace,
      maxAttempts: 3,
    });

    // 4. Assert Bindings
    await sdk.assertBinding({
      tenantCode: tenantCode,
      code: 'delayed-emails-binding',
      exchangeCode: 'email-events',
      queueCode: 'email-delayed',
      pattern: 'email.delayed.*',
      vnamespace: vnamespace,
      bindingType: 'classic',
    });

    await sdk.assertBinding({
      tenantCode: tenantCode,
      code: 'weekly-reports-binding',
      exchangeCode: 'email-events',
      queueCode: 'weekly-reports',
      pattern: 'report.weekly.*',
      vnamespace: vnamespace,
      bindingType: 'classic',
    });

    // ===== 5. Schedule Delayed Email (OneOff Scheduled Job) =====
    console.log('📅 Scheduling One-Off delayed welcome email (runs after 5s)...');
    const delayedEmailPayload: ScheduledEmail = {
      subject: 'Welcome to Acme Corp!',
      recipient: 'newuser@example.com',
      body: 'Thank you for signing up. Here is your onboarding guide.',
      type: 'one-off-welcome',
    };

    const oneOffJob = await sdk.createOneOffScheduledJob({
      tenantCode: tenantCode,
      targetType: 'exchange',
      targetCode: 'email-events',
      vnamespace: vnamespace,
      content: JSON.stringify(delayedEmailPayload),
      contentType: 'application/json',
      handler: 'email.send',
      runAfter: '5s',
      priority: 1,
      headers: { routing_key: 'email.delayed.welcome' },
    });
    console.log(`✅ OneOff Scheduled Job Created! ID: ${oneOffJob.id} | NextRunAt: ${oneOffJob.nextRunAt}`);

    // ===== 6. Schedule Recurring Weekly Report (Recurring Scheduled Job) =====
    console.log('🔄 Scheduling Recurring Weekly Report (runs every 10s)...');
    const recurringReportPayload: ScheduledEmail = {
      subject: 'Weekly Analytics & Performance Summary',
      recipient: 'executive-team@example.com',
      body: 'Attached is the weekly automated report.',
      type: 'recurring-weekly-report',
    };

    const recurringJob = await sdk.createRecurringScheduledJob({
      tenantCode: tenantCode,
      targetType: 'queue',
      targetCode: 'weekly-reports',
      vnamespace: vnamespace,
      content: JSON.stringify(recurringReportPayload),
      contentType: 'application/json',
      handler: 'report.generate',
      every: '10s',
      priority: 2,
    });
    console.log(`✅ Recurring Scheduled Job Created! ID: ${recurringJob.id} | Every: ${recurringJob.every} | NextRunAt: ${recurringJob.nextRunAt}`);

    // ===== 7. List Scheduled Jobs =====
    const listRes = await sdk.listScheduledJobs({
      tenantCode: tenantCode,
      vnamespace: vnamespace,
      pageSize: 10,
    });
    console.log(`📋 Currently Scheduled Jobs Count: ${listRes.entities.length}`);
    for (const j of listRes.entities) {
      console.log(`   - [${j.type}] ID: ${j.id} | Target: ${j.targetCode} (${j.targetType}) | NextRunAt: ${j.nextRunAt} | State: ${j.state}`);
    }

    // ===== 8. Start Worker to Process Scheduled Emails =====
    console.log('🚀 Starting worker to process scheduled email tasks...');

    sdk.createWorker({
      workerName: 'scheduled-email-worker-nodejs',
      intervalMs: 200,
      capacityPolicies: [
        {
          maxQueueMessages: 10,
          claimWorkFilter: {
            tenantCodes: [tenantCode],
            queueCodes: ['email-delayed', 'weekly-reports'],
          },
        },
      ],
      onMessage: async (claimed: ClaimedMessage, ack: AckCallback) => {
        try {
          const email: ScheduledEmail = JSON.parse(claimed.message.content);
          console.log('\n📩 [WORKER RECEIVED TASK]');
          console.log(`   Queue: ${claimed.message.queueId} | Handler: ${claimed.message.handler}`);
          console.log(`   To: ${email.recipient}`);
          console.log(`   Subject: ${email.subject}`);
          console.log(`   Type: ${email.type}\n`);
        } catch (e: any) {
          console.warn('⚠️ Failed to parse message content:', e.message);
        }
        await ack();
      },
    });

  } catch (err: any) {
    console.error('💥 Fatal error in main:', err.message);
  }
}

main();

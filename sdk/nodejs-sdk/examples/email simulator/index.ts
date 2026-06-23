import { DaedalusSDK } from '../../src/index';

const BATCH_SIZE = 500;
interface EmailMessage {
    messageId: string;
    companyId: string;
    emailType: 'transactional' | 'marketing' | 'report';
    from: string;
    to: string;
    subject: string;
    priority: 'urgent' | 'normal' | 'background';
    timestamp: number;
    size: number; // bytes
}

async function main() {
    const sdk = new DaedalusSDK({
        uri: 'http://localhost:4000',
        username: 'admin',
        password: 'admin'
    });

    await sdk.connect();

    // ===== SETUP: Per-Company Email Queues =====
    const companies = [
        { code: 'tienda-a', name: 'Big E-commerce', employees: 100000 },
        { code: 'tienda-b', name: 'Small Boutique', employees: 50 },
        { code: 'banco-c', name: 'Financial Bank', employees: 5000 },
        { code: 'empresa-x', name: 'Enterprise Corp', employees: 50000 }
    ];

    for (const company of companies) {
        // 1. Upsert company as tenant
        await sdk.assertTenant({
            code: company.code,
            name: company.name
        });

        // 2. Upsert exchange
        await sdk.assertExchange({
            tenantCode: company.code,
            code: 'email-events',
            name: 'Email Events',
            type: 'topic'
        });

        // 3. Upsert queues
        await sdk.assertQueue({
            tenantCode: company.code,
            code: 'email-transactional',
            name: 'Email Transactional',
            type: 'standard',
            state: 'active',
            vnamespace: 'default',
            allowDuplicated: false,
            maxAttempts: 3,
            priorityType: 'normal'
        });

        await sdk.assertQueue({
            tenantCode: company.code,
            code: 'email-marketing',
            name: 'Email Marketing',
            type: 'standard',
            state: 'active',
            vnamespace: 'default',
            allowDuplicated: false,
            maxAttempts: 3,
            priorityType: 'normal'
        });

        await sdk.assertQueue({
            tenantCode: company.code,
            code: 'email-report',
            name: 'Email Report',
            type: 'standard',
            state: 'active',
            vnamespace: 'default',
            allowDuplicated: false,
            maxAttempts: 3,
            priorityType: 'normal'
        });

        // 4. Bindings
        await sdk.assertBinding({
            code: 'transactional',
            tenantCode: company.code,
            exchangeCode: 'email-events',
            queueCode: 'email-transactional',
            pattern: 'transactional.*',
            vnamespace: 'default'
        });
        await sdk.assertBinding({
            code: 'marketing',
            tenantCode: company.code,
            exchangeCode: 'email-events',
            queueCode: 'email-marketing',
            pattern: 'marketing.*',
            vnamespace: 'default'
        });
        await sdk.assertBinding({
            code: 'report',
            tenantCode: company.code,
            exchangeCode: 'email-events',
            queueCode: 'email-report',
            pattern: 'report.*',
            vnamespace: 'default'
        });
    }

    // ===== PUBLISH: Black Friday Email Surge =====

    console.log('📧 CloudMail Pro: Black Friday Surge...');

    // Tienda A (Big): 5K transactional + 10K marketing
    console.log('🛍️  Tienda A: 5K confirmations + 10K Black Friday promos');

    await publishBatch(sdk, 5_000, (i) => {
        return {
            tenantCode: 'tienda-a',
            exchangeCode: 'email-events',
            routingKeyOrPatternOrQueueCode: 'transactional.order-confirmation',
            content: JSON.stringify({
                messageId: `tienda-a-trans-${i}`,
                companyId: 'tienda-a',
                emailType: 'transactional',
                from: 'orders@tienda-a.com',
                to: `customer-${i}@gmail.com`,
                subject: 'Order Confirmation #' + i,
                priority: 'urgent',
                timestamp: Date.now(),
                size: 5000
            }),
            vnamespace: 'default'
        };
    });

    await publishBatch(sdk, 10_000, (i) => {
        return {
            tenantCode: 'tienda-a',
            exchangeCode: 'email-events',
            routingKeyOrPatternOrQueueCode: 'marketing.black-friday',
            content: JSON.stringify({
                messageId: `tienda-a-mkt-${i}`,
                companyId: 'tienda-a',
                emailType: 'marketing',
                from: 'marketing@tienda-a.com',
                to: `customer-${i}@gmail.com`,
                subject: '🔥 BLACK FRIDAY: 50% OFF EVERYTHING',
                priority: 'normal',
                timestamp: Date.now(),
                size: 50000 // Marketing emails larger (images, etc)
            }),
            vnamespace: 'default'
        };
    });



    // Tienda B (Small): 5K transactional + 10K marketing (ISOLATED)
    console.log('🏪 Tienda B: 5K confirmations + 10K promos (ISOLATED from A)');

    await publishBatch(sdk, 5_000, (i) => {
        return {
            tenantCode: 'tienda-b',
            exchangeCode: 'email-events',
            routingKeyOrPatternOrQueueCode: 'transactional.order-confirmation',
            content: JSON.stringify({
                messageId: `tienda-b-trans-${i}`,
                companyId: 'tienda-b',
                emailType: 'transactional',
                from: 'orders@tienda-b.com',
                to: `customer-${i}@gmail.com`,
                subject: 'Order Confirmation',
                priority: 'urgent',
                timestamp: Date.now(),
                size: 5000
            }),
            vnamespace: 'default'
        };
    });

    await publishBatch(sdk, 10_000, (i) => {
        return {
            tenantCode: 'tienda-b',
            exchangeCode: 'email-events',
            routingKeyOrPatternOrQueueCode: 'marketing.black-friday-small',
            content: JSON.stringify({
                messageId: `tienda-b-mkt-${i}`,
                companyId: 'tienda-b',
                emailType: 'marketing',
                from: 'marketing@tienda-b.com',
                to: `customer-${i}@gmail.com`,
                subject: 'Black Friday Deals - Up to 40% Off',
                priority: 'normal',
                timestamp: Date.now(),
                size: 30000
            }),
            vnamespace: 'default'
        };
    });


    // Banco C: OTPs (ISOLATED)
    console.log('🏦 Banco C: 5K OTP emails (ISOLATED)');

    await publishBatch(sdk, 5_000, (i) => {
        return {
            tenantCode: 'banco-c',
            exchangeCode: 'email-events',
            routingKeyOrPatternOrQueueCode: 'transactional.otp',
            content: JSON.stringify({
                messageId: `banco-c-otp-${i}`,
                companyId: 'banco-c',
                emailType: 'transactional',
                from: 'security@banco-c.com',
                to: `client-${i}@banco-c.com`,
                subject: 'One-Time Password: ' + Math.random().toString().slice(2, 8),
                priority: 'urgent',
                timestamp: Date.now(),
                size: 2000
            }),
            vnamespace: 'default'
        };
    });


    // ===== WORKERS: Process emails =====

    let transactionalCount = 0;
    let marketingCount = 0;
    let reportCount = 0;

    // Worker 1: Transactional Processor (FAST, critical SLA)
    await sdk.createWorker({
        workerName: 'email-transactional-processor',
        intervalMs: 100,
        capacityPolicies: [
            {
                maxQueueMessages: 100,
                claimWorkFilter: {
                    tenantPatterns: ['*'],
                    queueCodes: ['email-transactional']
                }
            }
        ],
        onMessage: async (message, ack) => {
            const email = JSON.parse(message.message.content) as EmailMessage;
            console.log(
                `✉️  [TRANSACTIONAL] ${email.companyId}: ` +
                `To: ${email.to} | Subject: ${email.subject.slice(0, 30)}`
            );

            await new Promise(r => setTimeout(r, 10)); // Very fast
            transactionalCount++;
            console.log(`[Worker: email-transactional-processor] Processed message count: ${transactionalCount}`);
            return ack();
        }
    });

    // Worker 2: Marketing Processor
    await sdk.createWorker({
        workerName: 'email-marketing-processor',
        intervalMs: 100,
        capacityPolicies: [
            {
                maxQueueMessages: 100,
                claimWorkFilter: {
                    tenantPatterns: ['*'],
                    queueCodes: ['email-marketing']
                }
            }
        ],
        onMessage: async (message, ack) => {
            const email = JSON.parse(message.message.content) as EmailMessage;
            console.log(
                `📢 [MARKETING] ${email.companyId}: ` +
                `To: ${email.to} | Subject: ${email.subject.slice(0, 30)}`
            );

            await new Promise(r => setTimeout(r, 20)); // Slightly slower (bigger)
            marketingCount++;
            console.log(`[Worker: email-marketing-processor] Processed message count: ${marketingCount}`);
            return ack();
        }
    });

    // Worker 3: Reports/Analytics Processor (SLOW, background)
    await sdk.createWorker({
        workerName: 'email-report-processor',
        intervalMs: 1000,
        capacityPolicies: [
            {
                maxQueueMessages: 100,
                claimWorkFilter: {
                    tenantPatterns: ['*'],
                    queueCodes: ['email-report']
                }
            }
        ],
        onMessage: async (message, ack) => {
            const email = JSON.parse(message.message.content) as EmailMessage;
            console.log(
                `📊 [REPORT] ${email.companyId}: ` +
                `Archiving report: ${email.subject.slice(0, 30)}`
            );

            await new Promise(r => setTimeout(r, 500)); // Slow, background
            reportCount++;
            console.log(`[Worker: email-report-processor] Processed message count: ${reportCount}`);
            return ack();
        }
    });

    console.log('✅ CloudMail Pro running...');
    console.log('📧 Processing Black Friday surge across all companies');
}

async function publishBatch(
    sdk: DaedalusSDK,
    total: number,
    createMessage: (index: number) => {
        tenantCode: string;
        exchangeCode: string;
        routingKeyOrPatternOrQueueCode: string;
        content: string;
        vnamespace: string;
    }
) {
    for (let offset = 0; offset < total; offset += BATCH_SIZE) {
        const promises: Promise<unknown>[] = [];

        for (let i = offset; i < Math.min(offset + BATCH_SIZE, total); i++) {
            promises.push(sdk.publishMessage(createMessage(i)));
        }

        await Promise.all(promises);

        console.log(
            `Published ${Math.min(offset + BATCH_SIZE, total)}/${total}`
        );
    }
}

main().catch(err => {
    console.error('💥 Fatal error:', err);
});

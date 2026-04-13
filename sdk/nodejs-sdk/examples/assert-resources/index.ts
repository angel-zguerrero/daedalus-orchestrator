import { DaedalusSDK } from '../../src/index';

async function main() {
    const sdk = new DaedalusSDK({
        uri: 'http://localhost:4000',
        username: 'admin',
        password: 'admin'
    });

    await sdk.connect();

    // 1. Assert tenant
    const tenant = await sdk.assertTenant({
        code: 'my-tenant',
        name: 'My Tenant'
    });
    console.log('Tenant:', tenant);

    // 2. Assert exchange
    const exchange = await sdk.assertExchange({
        tenantCode: 'my-tenant',
        code: 'my-exchange',
        name: 'My Exchange',
        type: 'direct',
        vnamespace: 'default'
    });
    console.log('Exchange:', exchange);

    // 3. Assert queue
    const queue = await sdk.assertQueue({
        tenantCode: 'my-tenant',
        code: 'my-queue',
        name: 'My Queue',
        type: 'standard',
        state: 'active',
        vnamespace: 'default',
        maxAttempts: 3,
        maxQueueSize: 10000,
        priorityType: 'normal'
    });
    console.log('Queue:', queue);

    // 4. Assert binding (exchange → queue)
    const binding = await sdk.assertBinding({
        tenantCode: 'my-tenant',
        code: 'my-binding',
        exchangeCode: 'my-exchange',
        queueCode: 'my-queue',
        vnamespace: 'default',
        routingKey: 'my.routing.key',
        bindingType: 'classic'
    });
    console.log('Binding:', binding);

        await sdk.createWorker({
        workerName: 'Simple Node.js Worker 2',
        intervalMs: 500,
        capacityPolicies: [
            {
                maxQueueMessages: 0,
                claimWorkFilter: {
                }
            }
        ],
        onMessage: async (message, ack) => {
            console.log('👷 Processing message:', message);
            console.log('📝 Content:', message);
            
            // Simulate processing
            await new Promise(resolve => setTimeout(resolve, 10000));
            
            // Acknowledge the message after processing
            console.log('✅ Message processed, sending ACK...');
            await ack();
        }
    });

    // 5. Enqueue 1000 messages directly to the queue
    console.log('📤 Enqueueing 1000 messages...');
    const total = 1000;
    let succeeded = 0;
    for (let i = 0; i < total; i++) {
        const result = await sdk.enqueueMessage({
            tenantCode: 'my-tenant',
            queueCode: 'my-queue',
            vnamespace: 'default',
            content: JSON.stringify({ index: i, msg: `Hello from message ${i}` }),
            contentType: 'application/json',
            priority: 0,
            handler: 'my-handler'
        });
        succeeded++;
        if (succeeded % 100 === 0) {
            console.log(`  ✅ ${succeeded}/${total} messages enqueued (last id: ${result.messageId})`);
        }
    }
    console.log(`✅ Done. ${succeeded} messages enqueued to 'my-queue'.`);

    //await sdk.disconnect();



    console.log('✅ Worker is running. Press Ctrl+C to stop.');
    console.log('✅ All resources asserted successfully.');
}

main().catch(err => {
    console.error('💥 Fatal error:', err);
    process.exit(1);
});

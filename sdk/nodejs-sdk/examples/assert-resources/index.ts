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

    // 2. Assert queue
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

    // 3. Enqueue 1000 messages directly to the queue (batches of 50)
    console.log('📤 Enqueueing 1000 messages directly to the queue (batch size: 50)...');
    const total = 1000;
    const batchSize = 50;
    let succeeded = 0;
    for (let i = 0; i < total; i += batchSize) {
        const batch = Array.from({ length: Math.min(batchSize, total - i) }, (_, j) => {
            const idx = i + j;
            return sdk.enqueueMessage({
                tenantCode: 'my-tenant',
                queueCode: 'my-queue',
                vnamespace: 'default',
                content: JSON.stringify({ index: idx, msg: `Hello from message ${idx}` }),
                contentType: 'application/json',
                priority: 0,
                handler: 'my-handler'
            });
        });
        await Promise.all(batch);
        succeeded += batch.length;
        console.log(`  ✅ ${succeeded}/${total} messages enqueued`);
    }
    console.log(`✅ Done. ${succeeded} messages enqueued directly to 'my-queue'.`);

    //await sdk.disconnect();



    console.log('✅ Worker is running. Press Ctrl+C to stop.');
    console.log('✅ All resources asserted successfully.');
}

main().catch(err => {
    console.error('💥 Fatal error:', err);
    process.exit(1);
});

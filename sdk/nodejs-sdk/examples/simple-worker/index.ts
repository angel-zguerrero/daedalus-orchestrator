import { DaedalusSDK } from '../../src/index';

async function main() {
    const daedalusSDK = new DaedalusSDK({
        uri: 'http://localhost:4000',
        username: 'admin',
        password: 'admin'
    });

    await daedalusSDK.connect();

    await daedalusSDK.createWorker({
        workerName: 'Simple Node.js Worker 2',
        intervalMs: 500,
        capacityPolicies: [
            {
                maxQueueMessages: 10,
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

    console.log('✅ Worker is running. Press Ctrl+C to stop.');
}

main().catch(err => {
    console.error('💥 Fatal error:', err);
});

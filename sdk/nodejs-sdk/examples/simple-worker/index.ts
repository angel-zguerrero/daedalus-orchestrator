import { DaedalusSDK } from '../../src/index';
import * as si from 'systeminformation';

async function getSystemInfo(): Promise<Record<string, string>> {
    try {
        const cpu = await si.currentLoad();
        const mem = await si.mem();
        const disk = await si.fsSize();

        const info: Record<string, string> = {
            "CPU": cpu.currentLoad.toFixed(2),
            "Memory": ((mem.active / mem.total) * 100).toFixed(2),
            "Disk": disk[0].use.toFixed(2),
            "OS": String(process.platform),
            "Hostname": await si.osInfo().then(info => String(info.hostname))
        };
        return info;
    } catch (err) {
        console.error('❌ Error gathering system info:', err);
        return {
            "Error": "Failed to gather system info"
        };
    }
}

async function main() {
    const daedalusSDK = new DaedalusSDK({
        uri: 'http://localhost:4000',
        username: 'admin',
        password: 'admin'
    });

    await daedalusSDK.connect();

    await daedalusSDK.createWorker({
        workerName: 'Simple Node.js Worker 2',
        intervalMs: 10000,
        information: getSystemInfo,
        capacityPolicies: [
            {
                maxQueueMessages: 10,
                currentQueueMessages: 0,
                claimWorkFilter: {
                }
            }
        ],
        onMessage: async (message, ack) => {
            console.log('👷 Processing message:', message);
            console.log('📝 Content:', message);
            
            // Simulate processing
            await new Promise(resolve => setTimeout(resolve, 5000));
            
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

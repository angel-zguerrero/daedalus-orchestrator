import { DaedalusSDK as DaedalusJobWorker } from '../../src/index';

async function main() {
    const worker = new DaedalusJobWorker({
        endpoint: 'http://localhost:3000',
        username: 'admin',
        password: 'admin'
    });

    await worker.start();

}

main().catch(err => {
    console.error('Error running simple-worker:', err);
});

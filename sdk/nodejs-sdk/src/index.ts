import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import * as path from 'path';

export interface ClaimWorkFilter {
    tenantCodes?: string[];
    excludeTenantCodes?: string[];
    tenantPatterns?: string[];
    excludeTenantPatterns?: string[];
    vNamespaces?: string[];
    excludeVNamespaces?: string[];
    vNamespacePatterns?: string[];
    excludeVNamespacePatterns?: string[];
    queueCodes?: string[];
    excludeQueueCodes?: string[];
    queuePatterns?: string[];
    excludeQueuePatterns?: string[];
}

export interface ClaimWorkCapacityPolicy {
    maxQueueMessages: number;
    currentQueueMessages: number;
    claimWorkFilter?: ClaimWorkFilter;
}

export interface WorkerOptions {
    workerId: string;
    workerName: string;
    information?: Record<string, string> | (() => Promise<Record<string, string>> | Record<string, string>);
    capacityPolicies: ClaimWorkCapacityPolicy[];
    intervalMs?: number;
    onMessage?: (message: any) => Promise<void> | void;
}

export class DaedalusSDK {
    private jobWorkerClient: any;
    private authClient: any;
    private token: string | null = null;
    private jobWorkerProtoPath: string;
    private authProtoPath: string;

    constructor(private config: { uri: string, username: string, password: string }) {
        this.jobWorkerProtoPath = path.resolve(__dirname, '../../../server/internal/infrastructure/server/grpc/proto/definitions/jobworker.proto');
        this.authProtoPath = path.resolve(__dirname, '../../../server/internal/infrastructure/server/grpc/proto/definitions/auth.proto');
    }

    private loadProto(protoPath: string) {
        const packageDefinition = protoLoader.loadSync(protoPath, {
            keepCase: true,
            longs: String,
            enums: String,
            defaults: true,
            oneofs: true
        });
        return grpc.loadPackageDefinition(packageDefinition);
    }

    async connect() {
        const target = this.config.uri.replace('http://', '').replace('https://', '');

        // Load Auth Proto and create client
        const authProtoDescriptor = this.loadProto(this.authProtoPath) as any;
        this.authClient = new authProtoDescriptor.auth.AuthService(
            target,
            grpc.credentials.createInsecure()
        );

        // Perform Initial Login
        await this.login();

        // Load JobWorker Proto and create client
        const jobWorkerProtoDescriptor = this.loadProto(this.jobWorkerProtoPath) as any;
        this.jobWorkerClient = new jobWorkerProtoDescriptor.jobworker.JobWorkerService(
            target,
            grpc.credentials.createInsecure()
        );
    }

    private async login() {
        console.log(`🔐 Logging in as ${this.config.username}...`);
        try {
            const loginResponse = await new Promise<any>((resolve, reject) => {
                this.authClient.Login({
                    usernameOrEmail: this.config.username,
                    password: this.config.password
                }, (err: any, response: any) => {
                    if (err) return reject(err);
                    resolve(response);
                });
            });

            this.token = loginResponse.token;
            console.log('✅ Logged in successfully');
        } catch (err: any) {
            console.error('❌ Login failed:', err.message);
            throw err;
        }
    }

    async disconnect() {
        if (this.jobWorkerClient) {
            this.jobWorkerClient.close();
        }
        if (this.authClient) {
            this.authClient.close();
        }
    }

    private getMetadata(): grpc.Metadata {
        const metadata = new grpc.Metadata();
        if (this.token) {
            metadata.add('Authorization', `Bearer ${this.token}`);
        }
        return metadata;
    }

    async createWorker(options: WorkerOptions) {
        const {
            workerId,
            workerName,
            information,
            capacityPolicies,
            intervalMs = 10000,
            onMessage
        } = options;

        const run = async () => {
            try {
                if (!this.token) {
                    console.log('⚠️ Not authenticated. Attempting to log in...');
                    await this.login();
                }

                let currentInformation: Record<string, string> = {};
                if (typeof information === 'function') {
                    currentInformation = await information();
                } else if (information) {
                    currentInformation = information;
                }

                const request = {
                    workerID: workerId,
                    workerName: workerName,
                    information: currentInformation,
                    capacityPolicies: capacityPolicies
                };

                this.jobWorkerClient.ClaimWork(request, this.getMetadata(), async (err: any, response: any) => {
                    if (err) {
                        if (err.code === 16) { // UNAUTHENTICATED
                            console.warn('🔄 Session expired (Error 16). Refreshing token...');
                            this.token = null; // Clear token to trigger re-login on next run
                        } else {
                            console.error('❌ Error claiming work:', err.message);
                        }
                        return;
                    }

                    if (response && response.messages && response.messages.length > 0) {
                        console.log(`✅ Claimed ${response.messages.length} messages`);
                        if (onMessage) {
                            for (const msg of response.messages) {
                                try {
                                    await onMessage(msg);
                                } catch (processErr: any) {
                                    console.error('❌ Error processing message:', processErr.message);
                                }
                            }
                        }
                    }
                });
            } catch (err: any) {
                console.error('❌ Unexpected error in worker loop:', err.message);
            } finally {
                setTimeout(run, intervalMs);
            }
        };

        console.log(`🚀 Starting worker ${workerName} (${workerId}) with ${intervalMs}ms interval...`);
        run();
    }
}

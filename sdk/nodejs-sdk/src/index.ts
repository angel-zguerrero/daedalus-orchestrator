import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import * as path from 'path';
const crypto = require('crypto');

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

export interface QueueMessage {
    id: string;
    messageId: string;
    content: string;
    contentType: string;
    headers: Record<string, string>;
    queueId: string;
    priority: number;
    attempts: number;
    handler: string;
    parameters: Record<string, string>;
    vNamespace: string;
    createdAt: string;
}

export interface QueueMessageLease {
    id: string;
    queueMessageId: string;
    workerId: string;
    leaseUntil: string;
}

export interface ClaimedMessage {
    message: QueueMessage;
    lease: QueueMessageLease;
    tenantCode: string;
}

export interface AckCallback {
    (): Promise<void>;
}

export interface WorkerOptions {
    workerName: string;
    information?: Record<string, string> | (() => Promise<Record<string, string>> | Record<string, string>);
    capacityPolicies: ClaimWorkCapacityPolicy[];
    intervalMs?: number;
    onMessage?: (claimedMessage: ClaimedMessage, ack: AckCallback) => Promise<void> | void;
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

    async ackMessage(leaseID: string, tenantCode: string): Promise<void> {
        return new Promise((resolve, reject) => {
            this.jobWorkerClient.AckMessage(
                { leaseID, tenantCode },
                this.getMetadata(),
                (err: any, response: any) => {
                    if (err) {
                        console.error('❌ Failed to ack message:', err.message);
                        return reject(err);
                    }
                    if (!response.success) {
                        console.error('❌ Ack message failed:', response.message);
                        return reject(new Error(response.message));
                    }
                    console.log('✅ Message acknowledged successfully');
                    resolve();
                }
            );
        });
    }

    async createWorker(options: WorkerOptions) {
        const {
            workerName,
            information,
            capacityPolicies,
            intervalMs = 10000, // 10 seconds
            onMessage
        } = options;

        const workerId = `${crypto.randomUUID()}-${Date.now()}`;

        const run = async () => {
            try {
                if (!this.token) {
                    console.log('⚠️ Not authenticated. Attempting to log in...');
                    await this.login();
                }

                // Create bidirectional stream
                const call = this.jobWorkerClient.ClaimWork(this.getMetadata());

                console.log(`🔌 Opening bidirectional stream for worker ${workerId}...`);

                // Handle incoming messages from server
                call.on('data', async (streamMessage: any) => {
                    if (streamMessage.ack) {
                        console.log('✅ Connected to server:', streamMessage.ack.knowledge);
                    } else if (streamMessage.claimedMessage) {
                        const claimed = streamMessage.claimedMessage;
                        console.log(`📬 Received message: ${claimed.message.ID} from tenant ${claimed.tenantCode}`);

                        if (onMessage) {
                            try {
                                const claimedMessage: ClaimedMessage = {
                                    message: {
                                        id: claimed.message.ID,
                                        messageId: claimed.message.MessageID,
                                        content: claimed.message.Content,
                                        contentType: claimed.message.ContentType,
                                        headers: claimed.message.Headers || {},
                                        queueId: claimed.message.QueueID,
                                        priority: claimed.message.Priority,
                                        attempts: claimed.message.Attempts || 0,
                                        handler: claimed.message.Handler,
                                        parameters: claimed.message.Parameters || {},
                                        vNamespace: claimed.message.VNamespace,
                                        createdAt: claimed.message.CreatedAt
                                    },
                                    lease: {
                                        id: claimed.lease.ID,
                                        queueMessageId: claimed.lease.QueueMessageID,
                                        workerId: claimed.lease.WorkerID,
                                        leaseUntil: claimed.lease.LeaseUntil
                                    },
                                    tenantCode: claimed.tenantCode
                                };

                                const ackCallback: AckCallback = async () => {
                                    await this.ackMessage(claimed.lease.ID, claimed.tenantCode);
                                };

                                await onMessage(claimedMessage, ackCallback);
                            } catch (handlerError: any) {
                                console.error('❌ Error in onMessage handler:', handlerError.message);
                            }
                        }
                    }
                });

                call.on('error', (err: any) => {
                    if (err.code === 16) { // UNAUTHENTICATED
                        console.warn('🔄 Session expired (Error 16). Refreshing token...');
                        this.token = null;
                    } else if (err.code === 1) { // CANCELLED
                        console.log('🚫 Stream cancelled');
                    } else {
                        console.error('❌ Stream error:', err.message);
                    }
                });

                call.on('end', () => {
                    console.log('🔌 Stream ended, will reconnect...');
                });

                // Function to send claim request
                const sendClaimRequest = async () => {
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

                    call.write(request, (err: any) => {
                        if (err) {
                            console.error('❌ Error sending claim request:', err.message);
                        }
                    });
                };

                // Send initial claim request
                await sendClaimRequest();

                // Send claim requests periodically
                const claimInterval = setInterval(async () => {
                    await sendClaimRequest();
                }, intervalMs);

                // Wait for stream to end
                await new Promise<void>((resolve) => {
                    call.on('end', () => {
                        clearInterval(claimInterval);
                        resolve();
                    });
                    call.on('error', () => {
                        clearInterval(claimInterval);
                        resolve();
                    });
                });

            } catch (err: any) {
                console.error('❌ Unexpected error in worker loop:', err.message);
            }

            // Reconnect after delay
            console.log(`⏳ Reconnecting in ${intervalMs}ms...`);
            setTimeout(run, intervalMs);
        };

        console.log(`🚀 Starting worker ${workerName} (${workerId}) with ${intervalMs}ms interval...`);
        run();
    }
}

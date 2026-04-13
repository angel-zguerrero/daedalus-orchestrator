import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import * as path from 'path';
import * as si from 'systeminformation';
const crypto = require('crypto');

export async function getSystemInfo(): Promise<Record<string, string>> {
  try {
    const cpu = await si.currentLoad();
    const mem = await si.mem();
    const disk = await si.fsSize();

    const info: Record<string, string> = {
      "CPU": cpu.currentLoad.toFixed(2),
      "Memory": ((mem.active / mem.total) * 100).toFixed(2),
      "Disk": disk[0].use.toFixed(2),
      "OS": String(process.platform),
      "Hostname": await si.osInfo().then(i => String(i.hostname))
    };
    return info;
  } catch (err) {
    console.error('❌ Error gathering system info:', err);
    return {
      "Error": "Failed to gather system info"
    };
  }
}

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
  capacityPolicyIndexMatch: number;
}

export interface AckCallback {
  (): Promise<void>;
}

export interface WorkerOptions {
  workerName: string;
  capacityPolicies: ClaimWorkCapacityPolicy[];
  intervalMs?: number;
  onMessage?: (claimedMessage: ClaimedMessage, ack: AckCallback) => Promise<void> | void;
}

export interface AssertTenantInput {
  code: string;
  name: string;
}

export interface AssertExchangeInput {
  tenantCode: string;
  code: string;
  name: string;
  type: string;
  vnamespace?: string;
  headers?: Record<string, string>;
}

export interface AssertQueueInput {
  tenantCode: string;
  code: string;
  name: string;
  type?: string;
  state?: string;
  vnamespace?: string;
  defaultQueueMessageTTL?: number;
  defaultQueueMessageDelayTime?: number;
  queueExpires?: number;
  allowDuplicated?: boolean;
  maxAttempts?: number;
  maxQueueSize?: number;
  maxDeliveringMessages?: number;
  headers?: Record<string, string>;
}

export interface AssertBindingInput {
  tenantCode: string;
  code: string;
  exchangeCode: string;
  queueCode?: string;
  targetExchangeCode?: string;
  alternateExchangeCode?: string;
  vnamespace?: string;
  routingKey?: string;
  pattern?: string;
  xMatch?: string;
  bindingType?: string;
  targetExchangeType?: string;
  headers?: Record<string, string>;
}

export class DaedalusSDK {
  private jobWorkerClient: any;
  private authClient: any;
  private tenantClient: any;
  private exchangeClient: any;
  private queueClient: any;
  private bindingClient: any;
  private token: string | null = null;
  private jobWorkerProtoPath: string;
  private authProtoPath: string;
  private tenantProtoPath: string;
  private exchangeProtoPath: string;
  private queueProtoPath: string;
  private bindingProtoPath: string;

  constructor(private config: { uri: string, username: string, password: string }) {
    const protoBase = path.resolve(__dirname, '../../../server/internal/infrastructure/server/grpc/proto/definitions');
    this.jobWorkerProtoPath = path.join(protoBase, 'jobworker.proto');
    this.authProtoPath = path.join(protoBase, 'auth.proto');
    this.tenantProtoPath = path.join(protoBase, 'tenant.proto');
    this.exchangeProtoPath = path.join(protoBase, 'exchange.proto');
    this.queueProtoPath = path.join(protoBase, 'queue.proto');
    this.bindingProtoPath = path.join(protoBase, 'binding.proto');
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

    // Load resource management clients
    const tenantProtoDescriptor = this.loadProto(this.tenantProtoPath) as any;
    this.tenantClient = new tenantProtoDescriptor.tenant.TenantService(
      target,
      grpc.credentials.createInsecure()
    );

    const exchangeProtoDescriptor = this.loadProto(this.exchangeProtoPath) as any;
    this.exchangeClient = new exchangeProtoDescriptor.exchange.ExchangeService(
      target,
      grpc.credentials.createInsecure()
    );

    const queueProtoDescriptor = this.loadProto(this.queueProtoPath) as any;
    this.queueClient = new queueProtoDescriptor.queue.QueueService(
      target,
      grpc.credentials.createInsecure()
    );

    const bindingProtoDescriptor = this.loadProto(this.bindingProtoPath) as any;
    this.bindingClient = new bindingProtoDescriptor.binding.BindingService(
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
    if (this.tenantClient) {
      this.tenantClient.close();
    }
    if (this.exchangeClient) {
      this.exchangeClient.close();
    }
    if (this.queueClient) {
      this.queueClient.close();
    }
    if (this.bindingClient) {
      this.bindingClient.close();
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

  async assertTenant(input: AssertTenantInput): Promise<any> {
    return new Promise((resolve, reject) => {
      this.tenantClient.AssertTenant(
        { code: input.code, name: input.name },
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to assert tenant:', err.message);
            return reject(err);
          }
          console.log(`✅ Tenant asserted: ${input.code}`);
          resolve(response.result);
        }
      );
    });
  }

  async assertExchange(input: AssertExchangeInput): Promise<any> {
    return new Promise((resolve, reject) => {
      this.exchangeClient.CreateExchange(
        {
          tenantCode: input.tenantCode,
          code: input.code,
          name: input.name,
          type: input.type,
          vnamespace: input.vnamespace ?? '',
          headers: input.headers ?? {}
        },
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to assert exchange:', err.message);
            return reject(err);
          }
          console.log(`✅ Exchange asserted: ${input.code}`);
          resolve(response.result);
        }
      );
    });
  }

  async assertQueue(input: AssertQueueInput): Promise<any> {
    return new Promise((resolve, reject) => {
      this.queueClient.CreateQueue(
        {
          tenantCode: input.tenantCode,
          code: input.code,
          name: input.name,
          type: input.type ?? 'standard',
          state: input.state ?? 'active',
          vnamespace: input.vnamespace ?? '',
          defaultQueueMessageTTL: input.defaultQueueMessageTTL ?? 0,
          defaultQueueMessageDelayTime: input.defaultQueueMessageDelayTime ?? 0,
          queueExpires: input.queueExpires ?? 0,
          allowDuplicated: input.allowDuplicated ?? false,
          maxAttempts: input.maxAttempts ?? 0,
          maxQueueSize: input.maxQueueSize ?? 0,
          maxDeliveringMessages: input.maxDeliveringMessages ?? 0,
          headers: input.headers ?? {}
        },
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to assert queue:', err.message);
            return reject(err);
          }
          console.log(`✅ Queue asserted: ${input.code}`);
          resolve(response.result);
        }
      );
    });
  }

  async assertBinding(input: AssertBindingInput): Promise<any> {
    return new Promise((resolve, reject) => {
      this.bindingClient.CreateBinding(
        {
          tenantCode: input.tenantCode,
          code: input.code,
          exchangeCode: input.exchangeCode,
          queueCode: input.queueCode ?? '',
          targetExchangeCode: input.targetExchangeCode ?? '',
          alternateExchangeCode: input.alternateExchangeCode ?? '',
          vnamespace: input.vnamespace ?? '',
          routingKey: input.routingKey ?? '',
          pattern: input.pattern ?? '',
          xMatch: input.xMatch ?? '',
          bindingType: input.bindingType ?? 'classic',
          targetExchangeType: input.targetExchangeType ?? '',
          headers: input.headers ?? {}
        },
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to assert binding:', err.message);
            return reject(err);
          }
          console.log(`✅ Binding asserted: ${input.code}`);
          resolve(response.result);
        }
      );
    });
  }

  async createWorker(options: WorkerOptions) {

    const {
      workerName,
      capacityPolicies,
      intervalMs = 10000, // 10 seconds
      onMessage
    } = options;

    const workerId = `${crypto.randomUUID()}-${Date.now()}`;

    // Track in-flight message counts per capacity policy index.
    // Incremented when a message is received, decremented when ack'd.
    // Sent to the server on every heartbeat so it can enforce per-policy limits.
    const currentCounts = new Array(capacityPolicies.length).fill(0) as number[];

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
                  tenantCode: claimed.tenantCode,
                  capacityPolicyIndexMatch: claimed.capacityPolicyIndexMatch || 0
                };

                // Increment the in-flight count for the matched policy
                const policyIdx = claimedMessage.capacityPolicyIndexMatch;
                if (policyIdx >= 0 && policyIdx < currentCounts.length) {
                  currentCounts[policyIdx]++;
                }

                const ackCallback: AckCallback = async () => {
                  await this.ackMessage(claimed.lease.ID, claimed.tenantCode);
                  // Decrement the in-flight count after ack
                  if (policyIdx >= 0 && policyIdx < currentCounts.length) {
                    currentCounts[policyIdx] = Math.max(0, currentCounts[policyIdx] - 1);
                  }
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
          currentInformation = await getSystemInfo();

          const request = {
            workerID: workerId,
            workerName: workerName,
            information: currentInformation,
            capacityPolicies: capacityPolicies.map((p, i) => ({
              ...p,
              currentQueueMessages: currentCounts[i] ?? 0
            }))
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

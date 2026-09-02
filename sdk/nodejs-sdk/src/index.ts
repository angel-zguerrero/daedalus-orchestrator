import * as grpc from '@grpc/grpc-js';
import * as si from 'systeminformation';
const crypto = require('crypto');

import { AuthServiceClient } from './proto/auth_grpc_pb';
import { JobWorkerServiceClient } from './proto/jobworker_grpc_pb';
import { TenantServiceClient } from './proto/tenant_grpc_pb';
import { ExchangeServiceClient } from './proto/exchange_grpc_pb';
import { QueueServiceClient } from './proto/queue_grpc_pb';
import { BindingServiceClient } from './proto/binding_grpc_pb';

import { LoginRequest } from './proto/auth_pb';
import { ClaimWorkRequest, ClaimWorkCapacityPolicy as PBClaimWorkCapacityPolicy, ClaimWorkFilter as PBClaimWorkFilter, ClaimWorkStreamMessage, AckMessageRequest, BulkAckMessageRequest } from './proto/jobworker_pb';
import { AssertTenantRequest } from './proto/tenant_pb';
import { CreateExchangeRequest, PublishMessageRequest, QueueMessage as ExchangeQueueMessage, PublishStreamRequest } from './proto/exchange_pb';
import { CreateQueueRequest, EnqueueMessageRequest, EnqueueStreamRequest, BulkCreateQueueRequest, CreateQueueItem } from './proto/queue_pb';
import { CreateBindingRequest, BulkCreateBindingRequest, CreateBindingItem } from './proto/binding_pb';

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
  priorityType?: 'normal' | 'fair';
  desiredPriorityThresholds?: Record<number, number>;
  headers?: Record<string, string>;
}

export interface BulkAssertQueuesInput {
  tenantCode: string;
  queues: AssertQueueInput[];
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

export interface BulkAssertBindingsInput {
  tenantCode: string;
  bindings: AssertBindingInput[];
}

export interface EnqueueOptions {
  waitForConfirmation?: boolean; // default true
  timeoutMs?: number; // default 5000
}

export interface EnqueueResult {
  clientMessageId: string;
  messageId?: string;
  confirmed: boolean;
}

export interface EnqueueMessageInput {
  tenantCode: string;
  queueCode: string;
  content: string | Buffer;
  contentType?: string;
  vnamespace?: string;
  priority?: number;
  handler?: string;
  headers?: Record<string, string>;
  parameters?: Record<string, string>;
  options?: EnqueueOptions;
}

export interface PublishOptions {
  waitForConfirmation?: boolean; // default true
  timeoutMs?: number; // default 5000
}

export interface PublishResult {
  clientMessageId: string;
  confirmed: boolean;
  queueMessages?: Record<string, string>;
}

export interface PublishMessageInput {
  tenantCode: string;
  exchangeCode: string;
  routingKeyOrPatternOrQueueCode?: string;
  vnamespace?: string;
  content: string | Buffer;
  contentType?: string;
  priority?: number;
  handler?: string;
  headers?: Record<string, string>;
  parameters?: Record<string, string>;
  messageId?: string;
  options?: PublishOptions;
}

export interface SDKConfig {
  uri: string;
  username: string;
  password: string;
  autoReconnect?: boolean;
  maxReconnectAttempts?: number;
  reconnectIntervalMs?: number;
}

export class DaedalusSDK {
  private jobWorkerClient?: JobWorkerServiceClient;
  private authClient?: AuthServiceClient;
  private tenantClient?: TenantServiceClient;
  private exchangeClient?: ExchangeServiceClient;
  private queueClient?: QueueServiceClient;
  private bindingClient?: BindingServiceClient;
  private token: string | null = null;

  private publishStream: any = null;
  private publishPending: Map<string, { resolve: Function, reject: Function, timer: NodeJS.Timeout }> = new Map();

  private enqueueStream: any = null;
  private enqueuePending: Map<string, { resolve: Function, reject: Function, timer: NodeJS.Timeout }> = new Map();

  constructor(private config: SDKConfig) {
  }

  async connect() {
    const autoReconnect = this.config.autoReconnect !== false;
    const maxAttempts = this.config.maxReconnectAttempts;
    const interval = this.config.reconnectIntervalMs ?? 5000;

    let attempt = 0;
    while (true) {
      attempt++;
      try {
        await this._connectOnce();
        return;
      } catch (err: any) {
        if (!autoReconnect) {
          throw err;
        }
        if (maxAttempts !== undefined && attempt >= maxAttempts) {
          console.error(`❌ Connection failed after ${attempt} attempts. Giving up.`);
          throw err;
        }

        if (maxAttempts === undefined) {
          console.warn(`⚠️ Connection failed: ${err.message}. Reconnecting indefinitely (attempt ${attempt})...`);
        } else {
          console.warn(`⚠️ Connection failed: ${err.message}. Reconnecting (attempt ${attempt}/${maxAttempts})...`);
        }
        await new Promise(resolve => setTimeout(resolve, interval));
      }
    }
  }

  private async _connectOnce() {
    const target = this.config.uri.replace('http://', '').replace('https://', '');

    this.authClient = new AuthServiceClient(
      target,
      grpc.credentials.createInsecure()
    );

    await this.login();

    this.jobWorkerClient = new JobWorkerServiceClient(
      target,
      grpc.credentials.createInsecure()
    );

    this.tenantClient = new TenantServiceClient(
      target,
      grpc.credentials.createInsecure()
    );

    this.exchangeClient = new ExchangeServiceClient(
      target,
      grpc.credentials.createInsecure()
    );

    this.queueClient = new QueueServiceClient(
      target,
      grpc.credentials.createInsecure()
    );

    this.bindingClient = new BindingServiceClient(
      target,
      grpc.credentials.createInsecure()
    );
  }

  private async login() {
    console.log(`🔐 Logging in as ${this.config.username}...`);
    try {
      const req = new LoginRequest();
      req.setUsernameoremail(this.config.username);
      req.setPassword(this.config.password);

      const loginResponse = await new Promise<any>((resolve, reject) => {
        this.authClient!.login(req, (err: any, response: any) => {
          if (err) return reject(err);
          resolve(response.toObject());
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
    if (this.publishStream) {
      this.publishStream.end();
      this.publishStream = null;
    }
    if (this.enqueueStream) {
      this.enqueueStream.end();
      this.enqueueStream = null;
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
      const req = new AckMessageRequest();
      req.setLeaseid(leaseID);
      req.setTenantcode(tenantCode);

      this.jobWorkerClient!.ackMessage(
        req,
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to ack message:', err.message);
            return reject(err);
          }
          const respObj = response.toObject();
          if (!respObj.success) {
            console.error('❌ Ack message failed:', respObj.message);
            return reject(new Error(respObj.message));
          }
          resolve();
        }
      );
    });
  }

  async bulkAckMessages(leaseIDs: string[], tenantCode: string): Promise<void> {
    if (leaseIDs.length === 0) return;
    
    return new Promise((resolve, reject) => {
      const req = new BulkAckMessageRequest();
      req.setLeaseidsList(leaseIDs);
      req.setTenantcode(tenantCode);

      this.jobWorkerClient!.bulkAckMessages(
        req,
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to bulk ack messages:', err.message);
            return reject(err);
          }
          const respObj = response.toObject();
          if (!respObj.success) {
            console.error('❌ Bulk ack messages failed:', respObj.message);
            return reject(new Error(respObj.message));
          }
          resolve();
        }
      );
    });
  }

  async assertTenant(input: AssertTenantInput): Promise<any> {
    return new Promise((resolve, reject) => {
      const req = new AssertTenantRequest();
      req.setCode(input.code);
      req.setName(input.name);

      this.tenantClient!.assertTenant(
        req,
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to assert tenant:', err.message);
            return reject(err);
          }
          console.log(`✅ Tenant asserted: ${input.code}`);
          resolve(response.toObject().result);
        }
      );
    });
  }

  async assertExchange(input: AssertExchangeInput): Promise<any> {
    return new Promise((resolve, reject) => {
      const req = new CreateExchangeRequest();
      req.setTenantcode(input.tenantCode);
      req.setCode(input.code);
      req.setName(input.name);
      req.setType(input.type);
      req.setVnamespace(input.vnamespace ?? '');

      if (input.headers) {
        const headersMap = req.getHeadersMap();
        for (const [k, v] of Object.entries(input.headers)) {
          headersMap.set(k, v);
        }
      }

      this.exchangeClient!.createExchange(
        req,
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to assert exchange:', err.message);
            return reject(err);
          }
          console.log(`✅ Exchange asserted: ${input.code}`);
          resolve(response.toObject().result);
        }
      );
    });
  }

  async assertQueue(input: AssertQueueInput): Promise<any> {
    return new Promise((resolve, reject) => {
      const req = new CreateQueueRequest();
      req.setTenantcode(input.tenantCode);
      req.setCode(input.code);
      req.setName(input.name);
      req.setType(input.type ?? 'standard');
      req.setState(input.state ?? 'active');
      req.setVnamespace(input.vnamespace ?? '');
      req.setDefaultqueuemessagettl(input.defaultQueueMessageTTL ?? 0);
      req.setDefaultqueuemessagedelaytime(input.defaultQueueMessageDelayTime ?? 0);
      req.setQueueexpires(input.queueExpires ?? 0);
      req.setAllowduplicated(input.allowDuplicated ?? false);
      req.setMaxattempts(input.maxAttempts ?? 0);
      req.setMaxqueuesize(input.maxQueueSize ?? 0);
      req.setMaxdeliveringmessages(input.maxDeliveringMessages ?? 0);

      if (input.priorityType !== 'normal' && input.desiredPriorityThresholds) {
        const priorityMap = req.getDesiredprioritythresholdsMap();
        for (const [k, v] of Object.entries(input.desiredPriorityThresholds)) {
          priorityMap.set(Number(k), v);
        }
      }

      if (input.headers) {
        const headersMap = req.getHeadersMap();
        for (const [k, v] of Object.entries(input.headers)) {
          headersMap.set(k, v);
        }
      }

      this.queueClient!.createQueue(
        req,
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to assert queue:', err.message);
            return reject(err);
          }
          console.log(`✅ Queue asserted: ${input.code}`);
          resolve(response.toObject().result);
        }
      );
    });
  }

  async bulkAssertQueues(input: BulkAssertQueuesInput): Promise<any> {
    return new Promise((resolve, reject) => {
      const req = new BulkCreateQueueRequest();
      req.setTenantcode(input.tenantCode);

      for (const q of input.queues) {
        const item = new CreateQueueItem();
        item.setCode(q.code);
        item.setName(q.name);
        item.setType(q.type ?? 'standard');
        item.setState(q.state ?? 'active');
        item.setVnamespace(q.vnamespace ?? '');
        item.setDefaultqueuemessagettl(q.defaultQueueMessageTTL ?? 0);
        item.setDefaultqueuemessagedelaytime(q.defaultQueueMessageDelayTime ?? 0);
        item.setQueueexpires(q.queueExpires ?? 0);
        item.setAllowduplicated(q.allowDuplicated ?? false);
        item.setMaxattempts(q.maxAttempts ?? 0);
        item.setMaxqueuesize(q.maxQueueSize ?? 0);
        item.setMaxdeliveringmessages(q.maxDeliveringMessages ?? 0);

        if (q.priorityType !== 'normal' && q.desiredPriorityThresholds) {
          const priorityMap = item.getDesiredprioritythresholdsMap();
          for (const [k, v] of Object.entries(q.desiredPriorityThresholds)) {
            priorityMap.set(Number(k), v);
          }
        }

        if (q.headers) {
          const headersMap = item.getHeadersMap();
          for (const [k, v] of Object.entries(q.headers)) {
            headersMap.set(k, v);
          }
        }

        req.addQueues(item);
      }

      this.queueClient!.bulkCreateQueue(
        req,
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to bulk assert queues:', err.message);
            return reject(err);
          }
          console.log(`✅ Bulk Queues asserted: ${input.queues.length}`);
          resolve(response.toObject().resultList);
        }
      );
    });
  }

  private ensureEnqueueStream() {
    if (!this.enqueueStream) {
      this.enqueueStream = this.queueClient!.enqueueStream(this.getMetadata());
      this.enqueueStream.on('data', (response: any) => {
        const clientMessageId = response.getClientmessageid();
        const pending = this.enqueuePending.get(clientMessageId);
        if (pending) {
          clearTimeout(pending.timer);
          this.enqueuePending.delete(clientMessageId);
          if (!response.getConfirmed()) {
            pending.reject(new Error(response.getError() || 'Failed to enqueue message'));
          } else {
            pending.resolve({
              clientMessageId: clientMessageId,
              confirmed: true,
              messageId: response.getMessageid()
            });
          }
        }
      });
      this.enqueueStream.on('error', (err: any) => {
        console.error('EnqueueStream error:', err);
        this.enqueueStream = null;
      });
      this.enqueueStream.on('end', () => {
        this.enqueueStream = null;
      });
    }
  }

  private ensurePublishStream() {
    if (!this.publishStream) {
      this.publishStream = this.exchangeClient!.publishStream(this.getMetadata());
      this.publishStream.on('data', (response: any) => {
        const clientMessageId = response.getClientmessageid();
        const pending = this.publishPending.get(clientMessageId);
        if (pending) {
          clearTimeout(pending.timer);
          this.publishPending.delete(clientMessageId);
          if (!response.getConfirmed()) {
            pending.reject(new Error(response.getError() || 'Failed to publish message'));
          } else {
            pending.resolve({
              clientMessageId: clientMessageId,
              confirmed: true
            });
          }
        }
      });
      this.publishStream.on('error', (err: any) => {
        console.error('PublishStream error:', err);
        this.publishStream = null;
      });
      this.publishStream.on('end', () => {
        this.publishStream = null;
      });
    }
  }

  async enqueueMessage(input: EnqueueMessageInput): Promise<EnqueueResult> {
    const contentBytes = Buffer.isBuffer(input.content)
      ? input.content
      : Buffer.from(input.content);

    this.ensureEnqueueStream();

    const clientMessageId = crypto.randomUUID();
    const waitForConfirmation = input.options?.waitForConfirmation ?? true;
    const timeoutMs = input.options?.timeoutMs ?? 5000;

    const request = new EnqueueStreamRequest();
    request.setClientmessageid(clientMessageId);
    request.setTenantcode(input.tenantCode);
    request.setQueuecode(input.queueCode);
    request.setContent(contentBytes.toString('utf8'));
    request.setContenttype(input.contentType ?? 'text/plain');
    request.setVnamespace(input.vnamespace ?? '');
    request.setPriority(input.priority ?? 0);
    request.setHandler(input.handler ?? '');

    const headersMap = request.getHeadersMap();
    if (input.headers) {
      for (const [k, v] of Object.entries(input.headers)) {
        headersMap.set(k, v);
      }
    }

    const paramsMap = request.getParametersMap();
    if (input.parameters) {
      for (const [k, v] of Object.entries(input.parameters)) {
        paramsMap.set(k, v);
      }
    }

    if (!waitForConfirmation) {
      this.enqueueStream.write(request);
      return { clientMessageId, confirmed: false };
    }

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.enqueuePending.delete(clientMessageId);
        reject(new Error(`Enqueue confirmation timeout after ${timeoutMs}ms`));
      }, timeoutMs);

      this.enqueuePending.set(clientMessageId, { resolve, reject, timer });

      this.enqueueStream.write(request, (err: any) => {
        if (err) {
          clearTimeout(timer);
          this.enqueuePending.delete(clientMessageId);
          reject(err);
        }
      });
    });
  }

  async publishMessage(input: PublishMessageInput): Promise<PublishResult> {
    const contentBytes = Buffer.isBuffer(input.content)
      ? input.content
      : Buffer.from(input.content);

    this.ensurePublishStream();

    const clientMessageId = crypto.randomUUID();
    const waitForConfirmation = input.options?.waitForConfirmation ?? true;
    const timeoutMs = input.options?.timeoutMs ?? 5000;

    const request = new PublishStreamRequest();
    request.setClientmessageid(clientMessageId);
    request.setTenantcode(input.tenantCode);
    request.setExchangecode(input.exchangeCode);
    request.setRoutingkeyorpatternorqueuecode(input.routingKeyOrPatternOrQueueCode ?? '');
    request.setVnamespace(input.vnamespace ?? '');

    const message = new ExchangeQueueMessage();
    message.setMessageid(input.messageId ?? '');
    message.setHandler(input.handler ?? '');
    message.setPriority(input.priority ?? 0);
    message.setContenttype(input.contentType ?? 'text/plain');
    message.setContent(contentBytes);

    const headersMap = message.getHeadersMap();
    if (input.headers) {
      for (const [k, v] of Object.entries(input.headers)) {
        headersMap.set(k, v);
      }
    }

    const paramsMap = message.getParametersMap();
    if (input.parameters) {
      for (const [k, v] of Object.entries(input.parameters)) {
        paramsMap.set(k, v);
      }
    }

    request.setMessage(message);

    if (!waitForConfirmation) {
      this.publishStream.write(request);
      return { clientMessageId, confirmed: false };
    }

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.publishPending.delete(clientMessageId);
        reject(new Error(`Publish confirmation timeout after ${timeoutMs}ms`));
      }, timeoutMs);

      this.publishPending.set(clientMessageId, { resolve, reject, timer });

      this.publishStream.write(request, (err: any) => {
        if (err) {
          clearTimeout(timer);
          this.publishPending.delete(clientMessageId);
          reject(err);
        }
      });
    });
  }

  async assertBinding(input: AssertBindingInput): Promise<any> {
    return new Promise((resolve, reject) => {
      const req = new CreateBindingRequest();
      req.setTenantcode(input.tenantCode);
      req.setCode(input.code);
      req.setExchangecode(input.exchangeCode);
      req.setQueuecode(input.queueCode ?? '');
      req.setTargetexchangecode(input.targetExchangeCode ?? '');
      req.setAlternateexchangecode(input.alternateExchangeCode ?? '');
      req.setVnamespace(input.vnamespace ?? '');
      req.setRoutingkey(input.routingKey ?? '');
      req.setPattern(input.pattern ?? '');
      req.setXmatch(input.xMatch ?? '');
      req.setBindingtype(input.bindingType ?? 'classic');
      req.setTargetexchangetype(input.targetExchangeType ?? '');

      if (input.headers) {
        const headersMap = req.getHeadersMap();
        for (const [k, v] of Object.entries(input.headers)) {
          headersMap.set(k, v);
        }
      }

      this.bindingClient!.createBinding(
        req,
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to assert binding:', err.message);
            return reject(err);
          }
          console.log(`✅ Binding asserted: ${input.code}`);
          resolve(response.toObject().result);
        }
      );
    });
  }

  async bulkAssertBindings(input: BulkAssertBindingsInput): Promise<any> {
    return new Promise((resolve, reject) => {
      const req = new BulkCreateBindingRequest();
      req.setTenantcode(input.tenantCode);

      const bindingItems: CreateBindingItem[] = input.bindings.map(b => {
        const item = new CreateBindingItem();
        item.setCode(b.code);
        item.setExchangecode(b.exchangeCode);
        item.setQueuecode(b.queueCode ?? '');
        item.setTargetexchangecode(b.targetExchangeCode ?? '');
        item.setAlternateexchangecode(b.alternateExchangeCode ?? '');
        item.setVnamespace(b.vnamespace ?? '');
        item.setRoutingkey(b.routingKey ?? '');
        item.setPattern(b.pattern ?? '');
        item.setXmatch(b.xMatch ?? '');
        item.setBindingtype(b.bindingType ?? 'classic');
        item.setTargetexchangetype(b.targetExchangeType ?? '');

        if (b.headers) {
          const headersMap = item.getHeadersMap();
          for (const [k, v] of Object.entries(b.headers)) {
            headersMap.set(k, v);
          }
        }
        return item;
      });

      req.setBindingsList(bindingItems);

      this.bindingClient!.bulkCreateBinding(
        req,
        this.getMetadata(),
        (err: any, response: any) => {
          if (err) {
            console.error('❌ Failed to bulk assert bindings:', err.message);
            return reject(err);
          }
          console.log(`✅ Bulk Bindings asserted: ${input.bindings.length}`);
          resolve(response.toObject().resultsList);
        }
      );
    });
  }

  async createWorker(options: WorkerOptions) {

    const {
      workerName,
      capacityPolicies,
      intervalMs = 10000,
      onMessage
    } = options;

    const workerId = `${crypto.randomUUID()}-${Date.now()}`;
    const currentCounts = new Array(capacityPolicies.length).fill(0) as number[];

    let consecutiveFailures = 0;
    const run = async () => {
      const pendingAcks = new Map<string, { leaseIDs: string[], policyIndices: number[] }>();

      try {
        if (!this.token) {
          console.log('⚠️ Not authenticated. Attempting to log in...');
          await this.login();
        }

        const call = this.jobWorkerClient!.claimWork(this.getMetadata());

        console.log(`🔌 Opening bidirectional stream for worker ${workerId}...`);

        let connected = false;
        let streamResolved = false;
        let resolveStreamPromise: () => void;
        const streamPromise = new Promise<void>((resolve) => {
          resolveStreamPromise = () => {
            if (!streamResolved) {
              streamResolved = true;
              resolve();
            }
          };
        });

        call.on('data', async (streamMessage: ClaimWorkStreamMessage) => {
          if (streamMessage.hasAck()) {
            const ack = streamMessage.getAck()!;
            console.log('✅ Connected to server:', ack.getKnowledge());
            connected = true;
            consecutiveFailures = 0; // Reset failures on successful connection/ack
          } else if (streamMessage.hasClaimedmessage()) {
            const claimed = streamMessage.getClaimedmessage()!;

            if (onMessage) {
              try {
                const msgObj = claimed.getMessage()!;
                const leaseObj = claimed.getLease()!;

                const headersObj: Record<string, string> = {};
                msgObj.getHeadersMap().forEach((entry: string, key: string) => { headersObj[key] = entry; });

                const paramsObj: Record<string, string> = {};
                msgObj.getParametersMap().forEach((entry: string, key: string) => { paramsObj[key] = entry; });

                const claimedMessage: ClaimedMessage = {
                  message: {
                    id: msgObj.getId(),
                    messageId: msgObj.getMessageid(),
                    content: msgObj.getContent(),
                    contentType: msgObj.getContenttype(),
                    headers: headersObj,
                    queueId: msgObj.getQueueid(),
                    priority: msgObj.getPriority(),
                    attempts: msgObj.getAttempts() || 0,
                    handler: msgObj.getHandler(),
                    parameters: paramsObj,
                    vNamespace: msgObj.getVnamespace(),
                    createdAt: msgObj.getCreatedat()
                  },
                  lease: {
                    id: leaseObj.getId(),
                    queueMessageId: leaseObj.getQueuemessageid(),
                    workerId: leaseObj.getWorkerid(),
                    leaseUntil: leaseObj.getLeaseuntil()
                  },
                  tenantCode: claimed.getTenantcode(),
                  capacityPolicyIndexMatch: claimed.getCapacitypolicyindexmatch() || 0
                };

                const policyIdx = claimedMessage.capacityPolicyIndexMatch;
                if (policyIdx >= 0 && policyIdx < currentCounts.length) {
                  currentCounts[policyIdx]++;
                }

                const ackCallback: AckCallback = async () => {
                  let tenantData = pendingAcks.get(claimed.getTenantcode());
                  if (!tenantData) {
                    tenantData = { leaseIDs: [], policyIndices: [] };
                    pendingAcks.set(claimed.getTenantcode(), tenantData);
                  }
                  tenantData.leaseIDs.push(leaseObj.getId());
                  tenantData.policyIndices.push(policyIdx);

                  if (tenantData.leaseIDs.length >= 100) {
                    // Cannot await flushAcks here easily without hoisting or moving declaration,
                    // but we can just call it asynchronously as fire-and-forget from the callback's perspective.
                    // To avoid ReferenceError, we can use a setTimeout or just let the interval handle it,
                    // but calling it directly is fine if it's hoisted (which it isn't with const).
                    // We'll define a hoisted function or just wait for the 50ms interval.
                  }
                };

                Promise.resolve(onMessage(claimedMessage, ackCallback)).catch((handlerError: any) => {
                  console.error('❌ Error in onMessage handler:', handlerError.message);
                });
              } catch (handlerError: any) {
                console.error('❌ Error setting up onMessage handler:', handlerError.message);
              }
            }
          }
        });

        call.on('error', (err: any) => {
          if (err.code === 16) {
            console.warn('🔄 Session expired (Error 16). Refreshing token...');
            this.token = null;
          } else if (err.code === 1) {
            console.log('🚫 Stream cancelled');
          } else {
            console.error('❌ Stream error:', err.message);
          }
          resolveStreamPromise();
        });

        call.on('end', () => {
          console.log('🔌 Stream ended, will reconnect...');
          resolveStreamPromise();
        });

        // Function to send claim request
        let currentInformation: Record<string, string> = await getSystemInfo();

        // Update system information every 15 seconds to avoid heavy OS polling
        const sysInfoInterval = setInterval(async () => {
          try {
            currentInformation = await getSystemInfo();
          } catch (err) {
            console.error('Failed to update system info:', err);
          }
        }, 15000);

        const sendClaimRequest = () => {
          const req = new ClaimWorkRequest();
          req.setWorkerid(workerId);
          req.setWorkername(workerName);

          const infoMap = req.getInformationMap();
          for (const [k, v] of Object.entries(currentInformation)) {
            infoMap.set(k, v);
          }

          capacityPolicies.forEach((p, i) => {
            const cp = new PBClaimWorkCapacityPolicy();
            cp.setMaxqueuemessages(p.maxQueueMessages);
            cp.setCurrentqueuemessages(currentCounts[i] ?? 0);

            if (p.claimWorkFilter) {
              const f = new PBClaimWorkFilter();
              if (p.claimWorkFilter.tenantCodes) p.claimWorkFilter.tenantCodes.forEach(v => f.addTenantcodes(v));
              if (p.claimWorkFilter.excludeTenantCodes) p.claimWorkFilter.excludeTenantCodes.forEach(v => f.addExcludetenantcodes(v));
              if (p.claimWorkFilter.tenantPatterns) p.claimWorkFilter.tenantPatterns.forEach(v => f.addTenantpatterns(v));
              if (p.claimWorkFilter.excludeTenantPatterns) p.claimWorkFilter.excludeTenantPatterns.forEach(v => f.addExcludetenantpatterns(v));
              if (p.claimWorkFilter.vNamespaces) p.claimWorkFilter.vNamespaces.forEach(v => f.addVnamespaces(v));
              if (p.claimWorkFilter.excludeVNamespaces) p.claimWorkFilter.excludeVNamespaces.forEach(v => f.addExcludevnamespaces(v));
              if (p.claimWorkFilter.vNamespacePatterns) p.claimWorkFilter.vNamespacePatterns.forEach(v => f.addVnamespacepatterns(v));
              if (p.claimWorkFilter.excludeVNamespacePatterns) p.claimWorkFilter.excludeVNamespacePatterns.forEach(v => f.addExcludevnamespacepatterns(v));
              if (p.claimWorkFilter.queueCodes) p.claimWorkFilter.queueCodes.forEach(v => f.addQueuecodes(v));
              if (p.claimWorkFilter.excludeQueueCodes) p.claimWorkFilter.excludeQueueCodes.forEach(v => f.addExcludequeuecodes(v));
              if (p.claimWorkFilter.queuePatterns) p.claimWorkFilter.queuePatterns.forEach(v => f.addQueuepatterns(v));
              if (p.claimWorkFilter.excludeQueuePatterns) p.claimWorkFilter.excludeQueuePatterns.forEach(v => f.addExcludequeuepatterns(v));
              cp.setClaimworkfilter(f);
            }

            req.addCapacitypolicies(cp);
          });

          call.write(req);
        };

        let claimRequestPending = false;
        const triggerClaimRequest = () => {
          if (claimRequestPending) return;
          claimRequestPending = true;
          setTimeout(() => {
            claimRequestPending = false;
            if (connected) {
              sendClaimRequest();
            }
          }, 50); // 50ms debounce
        };

        const flushAcks = async () => {
          if (pendingAcks.size === 0) return;
          
          const currentAcks = new Map(pendingAcks);
          pendingAcks.clear();

          for (const [tenantCode, data] of currentAcks.entries()) {
            try {
              await this.bulkAckMessages(data.leaseIDs, tenantCode);
              for (const policyIdx of data.policyIndices) {
                if (policyIdx >= 0 && policyIdx < currentCounts.length) {
                  currentCounts[policyIdx] = Math.max(0, currentCounts[policyIdx] - 1);
                }
              }
              triggerClaimRequest();
            } catch (err: any) {
              console.error(`❌ Failed to bulk ack for tenant ${tenantCode}:`, err.message);
            }
          }
        };

        // Send initial claim request
        sendClaimRequest();

        // Send claim requests periodically
        const claimInterval = setInterval(() => {
          sendClaimRequest();
        }, intervalMs);

        // Flush ACKs periodically
        const ackFlushInterval = setInterval(flushAcks, 50);

        await streamPromise;

        clearInterval(claimInterval);
        clearInterval(sysInfoInterval);
        clearInterval(ackFlushInterval);

        if (!connected) {
          throw new Error('Failed to establish stream connection');
        }

      } catch (err: any) {
        console.error('❌ Unexpected error in worker loop:', err.message);
        consecutiveFailures++;

        const autoReconnect = this.config.autoReconnect !== false;
        const maxAttempts = this.config.maxReconnectAttempts;
        if (!autoReconnect) {
          console.log('🛑 Auto reconnect is disabled. Worker will exit.');
          return;
        }
        if (maxAttempts !== undefined && consecutiveFailures >= maxAttempts) {
          console.error(`❌ Worker reconnection failed after ${consecutiveFailures} attempts. Worker will exit.`);
          return;
        }

        if (maxAttempts === undefined) {
          console.warn(`⚠️ Worker connection failed. Reconnecting indefinitely (attempt ${consecutiveFailures})...`);
        } else {
          console.warn(`⚠️ Worker connection failed. Reconnecting (attempt ${consecutiveFailures}/${maxAttempts})...`);
        }
      }

      const autoReconnect = this.config.autoReconnect !== false;
      if (!autoReconnect) {
        console.log('🛑 Auto reconnect is disabled. Worker will exit.');
        return;
      }

      console.log(`⏳ Reconnecting in ${intervalMs}ms...`);
      setTimeout(run, intervalMs);
    };

    console.log(`🚀 Starting worker ${workerName} (${workerId}) with ${intervalMs}ms interval...`);
    run();
  }
}

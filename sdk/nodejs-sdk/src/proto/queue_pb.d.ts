// package: queue
// file: queue.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

export class CreateQueueRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): CreateQueueRequest;
    getCode(): string;
    setCode(value: string): CreateQueueRequest;
    getName(): string;
    setName(value: string): CreateQueueRequest;
    getType(): string;
    setType(value: string): CreateQueueRequest;
    getState(): string;
    setState(value: string): CreateQueueRequest;
    getVnamespace(): string;
    setVnamespace(value: string): CreateQueueRequest;
    getDefaultqueuemessagettl(): number;
    setDefaultqueuemessagettl(value: number): CreateQueueRequest;
    getDefaultqueuemessagedelaytime(): number;
    setDefaultqueuemessagedelaytime(value: number): CreateQueueRequest;
    getQueueexpires(): number;
    setQueueexpires(value: number): CreateQueueRequest;
    getAllowduplicated(): boolean;
    setAllowduplicated(value: boolean): CreateQueueRequest;
    getMaxattempts(): number;
    setMaxattempts(value: number): CreateQueueRequest;

    getDesiredprioritythresholdsMap(): jspb.Map<number, number>;
    clearDesiredprioritythresholdsMap(): void;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;
    getDeadletterexchangeid(): string;
    setDeadletterexchangeid(value: string): CreateQueueRequest;
    getDeadletterexchangeroutingkeyorpattern(): string;
    setDeadletterexchangeroutingkeyorpattern(value: string): CreateQueueRequest;
    getMaxqueuesize(): number;
    setMaxqueuesize(value: number): CreateQueueRequest;
    getMaxdeliveringmessages(): number;
    setMaxdeliveringmessages(value: number): CreateQueueRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateQueueRequest.AsObject;
    static toObject(includeInstance: boolean, msg: CreateQueueRequest): CreateQueueRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateQueueRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateQueueRequest;
    static deserializeBinaryFromReader(message: CreateQueueRequest, reader: jspb.BinaryReader): CreateQueueRequest;
}

export namespace CreateQueueRequest {
    export type AsObject = {
        tenantcode: string,
        code: string,
        name: string,
        type: string,
        state: string,
        vnamespace: string,
        defaultqueuemessagettl: number,
        defaultqueuemessagedelaytime: number,
        queueexpires: number,
        allowduplicated: boolean,
        maxattempts: number,

        desiredprioritythresholdsMap: Array<[number, number]>,

        headersMap: Array<[string, string]>,
        deadletterexchangeid: string,
        deadletterexchangeroutingkeyorpattern: string,
        maxqueuesize: number,
        maxdeliveringmessages: number,
    }
}

export class CreateQueueResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): CreateQueueResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): Queue | undefined;
    setResult(value?: Queue): CreateQueueResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateQueueResponse.AsObject;
    static toObject(includeInstance: boolean, msg: CreateQueueResponse): CreateQueueResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateQueueResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateQueueResponse;
    static deserializeBinaryFromReader(message: CreateQueueResponse, reader: jspb.BinaryReader): CreateQueueResponse;
}

export namespace CreateQueueResponse {
    export type AsObject = {
        message: string,
        result?: Queue.AsObject,
    }
}

export class BulkCreateQueueRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): BulkCreateQueueRequest;
    clearQueuesList(): void;
    getQueuesList(): Array<CreateQueueItem>;
    setQueuesList(value: Array<CreateQueueItem>): BulkCreateQueueRequest;
    addQueues(value?: CreateQueueItem, index?: number): CreateQueueItem;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): BulkCreateQueueRequest.AsObject;
    static toObject(includeInstance: boolean, msg: BulkCreateQueueRequest): BulkCreateQueueRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: BulkCreateQueueRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): BulkCreateQueueRequest;
    static deserializeBinaryFromReader(message: BulkCreateQueueRequest, reader: jspb.BinaryReader): BulkCreateQueueRequest;
}

export namespace BulkCreateQueueRequest {
    export type AsObject = {
        tenantcode: string,
        queuesList: Array<CreateQueueItem.AsObject>,
    }
}

export class CreateQueueItem extends jspb.Message { 
    getCode(): string;
    setCode(value: string): CreateQueueItem;
    getName(): string;
    setName(value: string): CreateQueueItem;
    getType(): string;
    setType(value: string): CreateQueueItem;
    getState(): string;
    setState(value: string): CreateQueueItem;
    getVnamespace(): string;
    setVnamespace(value: string): CreateQueueItem;
    getDefaultqueuemessagettl(): number;
    setDefaultqueuemessagettl(value: number): CreateQueueItem;
    getDefaultqueuemessagedelaytime(): number;
    setDefaultqueuemessagedelaytime(value: number): CreateQueueItem;
    getQueueexpires(): number;
    setQueueexpires(value: number): CreateQueueItem;
    getAllowduplicated(): boolean;
    setAllowduplicated(value: boolean): CreateQueueItem;
    getMaxattempts(): number;
    setMaxattempts(value: number): CreateQueueItem;

    getDesiredprioritythresholdsMap(): jspb.Map<number, number>;
    clearDesiredprioritythresholdsMap(): void;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;
    getDeadletterexchangeid(): string;
    setDeadletterexchangeid(value: string): CreateQueueItem;
    getDeadletterexchangeroutingkeyorpattern(): string;
    setDeadletterexchangeroutingkeyorpattern(value: string): CreateQueueItem;
    getMaxqueuesize(): number;
    setMaxqueuesize(value: number): CreateQueueItem;
    getMaxdeliveringmessages(): number;
    setMaxdeliveringmessages(value: number): CreateQueueItem;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateQueueItem.AsObject;
    static toObject(includeInstance: boolean, msg: CreateQueueItem): CreateQueueItem.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateQueueItem, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateQueueItem;
    static deserializeBinaryFromReader(message: CreateQueueItem, reader: jspb.BinaryReader): CreateQueueItem;
}

export namespace CreateQueueItem {
    export type AsObject = {
        code: string,
        name: string,
        type: string,
        state: string,
        vnamespace: string,
        defaultqueuemessagettl: number,
        defaultqueuemessagedelaytime: number,
        queueexpires: number,
        allowduplicated: boolean,
        maxattempts: number,

        desiredprioritythresholdsMap: Array<[number, number]>,

        headersMap: Array<[string, string]>,
        deadletterexchangeid: string,
        deadletterexchangeroutingkeyorpattern: string,
        maxqueuesize: number,
        maxdeliveringmessages: number,
    }
}

export class BulkCreateQueueResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): BulkCreateQueueResponse;
    clearResultList(): void;
    getResultList(): Array<Queue>;
    setResultList(value: Array<Queue>): BulkCreateQueueResponse;
    addResult(value?: Queue, index?: number): Queue;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): BulkCreateQueueResponse.AsObject;
    static toObject(includeInstance: boolean, msg: BulkCreateQueueResponse): BulkCreateQueueResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: BulkCreateQueueResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): BulkCreateQueueResponse;
    static deserializeBinaryFromReader(message: BulkCreateQueueResponse, reader: jspb.BinaryReader): BulkCreateQueueResponse;
}

export namespace BulkCreateQueueResponse {
    export type AsObject = {
        message: string,
        resultList: Array<Queue.AsObject>,
    }
}

export class GetQueueRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): GetQueueRequest;
    getCode(): string;
    setCode(value: string): GetQueueRequest;
    getVnamespace(): string;
    setVnamespace(value: string): GetQueueRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetQueueRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetQueueRequest): GetQueueRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetQueueRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetQueueRequest;
    static deserializeBinaryFromReader(message: GetQueueRequest, reader: jspb.BinaryReader): GetQueueRequest;
}

export namespace GetQueueRequest {
    export type AsObject = {
        tenantcode: string,
        code: string,
        vnamespace: string,
    }
}

export class GetQueueResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): GetQueueResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): Queue | undefined;
    setResult(value?: Queue): GetQueueResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetQueueResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetQueueResponse): GetQueueResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetQueueResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetQueueResponse;
    static deserializeBinaryFromReader(message: GetQueueResponse, reader: jspb.BinaryReader): GetQueueResponse;
}

export namespace GetQueueResponse {
    export type AsObject = {
        message: string,
        result?: Queue.AsObject,
    }
}

export class GetQueuesRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): GetQueuesRequest;
    getQ(): string;
    setQ(value: string): GetQueuesRequest;
    getCursor(): string;
    setCursor(value: string): GetQueuesRequest;
    getPagesize(): number;
    setPagesize(value: number): GetQueuesRequest;
    getVnamespace(): string;
    setVnamespace(value: string): GetQueuesRequest;
    getIncludeheaders(): boolean;
    setIncludeheaders(value: boolean): GetQueuesRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetQueuesRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetQueuesRequest): GetQueuesRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetQueuesRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetQueuesRequest;
    static deserializeBinaryFromReader(message: GetQueuesRequest, reader: jspb.BinaryReader): GetQueuesRequest;
}

export namespace GetQueuesRequest {
    export type AsObject = {
        tenantcode: string,
        q: string,
        cursor: string,
        pagesize: number,
        vnamespace: string,
        includeheaders: boolean,
    }
}

export class Queue extends jspb.Message { 
    getId(): string;
    setId(value: string): Queue;
    getCode(): string;
    setCode(value: string): Queue;
    getName(): string;
    setName(value: string): Queue;
    getType(): string;
    setType(value: string): Queue;
    getState(): string;
    setState(value: string): Queue;
    getVnamespace(): string;
    setVnamespace(value: string): Queue;
    getCreatedat(): string;
    setCreatedat(value: string): Queue;
    getUpdatedat(): string;
    setUpdatedat(value: string): Queue;
    getDefaultqueuemessagettl(): number;
    setDefaultqueuemessagettl(value: number): Queue;
    getDefaultqueuemessagedelaytime(): number;
    setDefaultqueuemessagedelaytime(value: number): Queue;
    getQueueexpires(): number;
    setQueueexpires(value: number): Queue;
    getExpireat(): string;
    setExpireat(value: string): Queue;
    getAllowduplicated(): boolean;
    setAllowduplicated(value: boolean): Queue;
    getMaxattempts(): number;
    setMaxattempts(value: number): Queue;

    getDesiredprioritythresholdsMap(): jspb.Map<number, number>;
    clearDesiredprioritythresholdsMap(): void;

    getPrioritythresholdsMap(): jspb.Map<number, number>;
    clearPrioritythresholdsMap(): void;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;
    getDeadletterexchangeid(): string;
    setDeadletterexchangeid(value: string): Queue;
    getDeadletterexchangeroutingkeyorpattern(): string;
    setDeadletterexchangeroutingkeyorpattern(value: string): Queue;
    getMessagescount(): number;
    setMessagescount(value: number): Queue;
    getMaxqueuesize(): number;
    setMaxqueuesize(value: number): Queue;
    getNodeschedulersupervisorid(): string;
    setNodeschedulersupervisorid(value: string): Queue;
    getNodeschedulersupervisorcode(): string;
    setNodeschedulersupervisorcode(value: string): Queue;
    getNodeschedulersupervisorname(): string;
    setNodeschedulersupervisorname(value: string): Queue;
    getNodeschedulerqueuesupervisionstate(): string;
    setNodeschedulerqueuesupervisionstate(value: string): Queue;
    getMaxdeliveringmessages(): number;
    setMaxdeliveringmessages(value: number): Queue;
    getCurrentdeliveringmessages(): number;
    setCurrentdeliveringmessages(value: number): Queue;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Queue.AsObject;
    static toObject(includeInstance: boolean, msg: Queue): Queue.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Queue, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Queue;
    static deserializeBinaryFromReader(message: Queue, reader: jspb.BinaryReader): Queue;
}

export namespace Queue {
    export type AsObject = {
        id: string,
        code: string,
        name: string,
        type: string,
        state: string,
        vnamespace: string,
        createdat: string,
        updatedat: string,
        defaultqueuemessagettl: number,
        defaultqueuemessagedelaytime: number,
        queueexpires: number,
        expireat: string,
        allowduplicated: boolean,
        maxattempts: number,

        desiredprioritythresholdsMap: Array<[number, number]>,

        prioritythresholdsMap: Array<[number, number]>,

        headersMap: Array<[string, string]>,
        deadletterexchangeid: string,
        deadletterexchangeroutingkeyorpattern: string,
        messagescount: number,
        maxqueuesize: number,
        nodeschedulersupervisorid: string,
        nodeschedulersupervisorcode: string,
        nodeschedulersupervisorname: string,
        nodeschedulerqueuesupervisionstate: string,
        maxdeliveringmessages: number,
        currentdeliveringmessages: number,
    }
}

export class QueueFindResult extends jspb.Message { 
    clearEntitiesList(): void;
    getEntitiesList(): Array<Queue>;
    setEntitiesList(value: Array<Queue>): QueueFindResult;
    addEntities(value?: Queue, index?: number): Queue;
    getCursor(): string;
    setCursor(value: string): QueueFindResult;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): QueueFindResult.AsObject;
    static toObject(includeInstance: boolean, msg: QueueFindResult): QueueFindResult.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: QueueFindResult, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): QueueFindResult;
    static deserializeBinaryFromReader(message: QueueFindResult, reader: jspb.BinaryReader): QueueFindResult;
}

export namespace QueueFindResult {
    export type AsObject = {
        entitiesList: Array<Queue.AsObject>,
        cursor: string,
    }
}

export class GetQueuesResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): GetQueuesResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): QueueFindResult | undefined;
    setResult(value?: QueueFindResult): GetQueuesResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetQueuesResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetQueuesResponse): GetQueuesResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetQueuesResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetQueuesResponse;
    static deserializeBinaryFromReader(message: GetQueuesResponse, reader: jspb.BinaryReader): GetQueuesResponse;
}

export namespace GetQueuesResponse {
    export type AsObject = {
        message: string,
        result?: QueueFindResult.AsObject,
    }
}

export class DeleteQueueRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): DeleteQueueRequest;
    getCode(): string;
    setCode(value: string): DeleteQueueRequest;
    getVnamespace(): string;
    setVnamespace(value: string): DeleteQueueRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteQueueRequest.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteQueueRequest): DeleteQueueRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteQueueRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteQueueRequest;
    static deserializeBinaryFromReader(message: DeleteQueueRequest, reader: jspb.BinaryReader): DeleteQueueRequest;
}

export namespace DeleteQueueRequest {
    export type AsObject = {
        tenantcode: string,
        code: string,
        vnamespace: string,
    }
}

export class DeleteQueueResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): DeleteQueueResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteQueueResponse.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteQueueResponse): DeleteQueueResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteQueueResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteQueueResponse;
    static deserializeBinaryFromReader(message: DeleteQueueResponse, reader: jspb.BinaryReader): DeleteQueueResponse;
}

export namespace DeleteQueueResponse {
    export type AsObject = {
        message: string,
    }
}

export class EnqueueMessageRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): EnqueueMessageRequest;
    getQueuecode(): string;
    setQueuecode(value: string): EnqueueMessageRequest;
    getVnamespace(): string;
    setVnamespace(value: string): EnqueueMessageRequest;
    getContent(): string;
    setContent(value: string): EnqueueMessageRequest;
    getContenttype(): string;
    setContenttype(value: string): EnqueueMessageRequest;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;
    getPriority(): number;
    setPriority(value: number): EnqueueMessageRequest;
    getHandler(): string;
    setHandler(value: string): EnqueueMessageRequest;

    getParametersMap(): jspb.Map<string, string>;
    clearParametersMap(): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): EnqueueMessageRequest.AsObject;
    static toObject(includeInstance: boolean, msg: EnqueueMessageRequest): EnqueueMessageRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: EnqueueMessageRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): EnqueueMessageRequest;
    static deserializeBinaryFromReader(message: EnqueueMessageRequest, reader: jspb.BinaryReader): EnqueueMessageRequest;
}

export namespace EnqueueMessageRequest {
    export type AsObject = {
        tenantcode: string,
        queuecode: string,
        vnamespace: string,
        content: string,
        contenttype: string,

        headersMap: Array<[string, string]>,
        priority: number,
        handler: string,

        parametersMap: Array<[string, string]>,
    }
}

export class EnqueueMessageResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): EnqueueMessageResponse;
    getMessageid(): string;
    setMessageid(value: string): EnqueueMessageResponse;

    getResultMap(): jspb.Map<string, string>;
    clearResultMap(): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): EnqueueMessageResponse.AsObject;
    static toObject(includeInstance: boolean, msg: EnqueueMessageResponse): EnqueueMessageResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: EnqueueMessageResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): EnqueueMessageResponse;
    static deserializeBinaryFromReader(message: EnqueueMessageResponse, reader: jspb.BinaryReader): EnqueueMessageResponse;
}

export namespace EnqueueMessageResponse {
    export type AsObject = {
        message: string,
        messageid: string,

        resultMap: Array<[string, string]>,
    }
}

export class EnqueueStreamRequest extends jspb.Message { 
    getClientmessageid(): string;
    setClientmessageid(value: string): EnqueueStreamRequest;
    getTenantcode(): string;
    setTenantcode(value: string): EnqueueStreamRequest;
    getQueuecode(): string;
    setQueuecode(value: string): EnqueueStreamRequest;
    getVnamespace(): string;
    setVnamespace(value: string): EnqueueStreamRequest;
    getContent(): string;
    setContent(value: string): EnqueueStreamRequest;
    getContenttype(): string;
    setContenttype(value: string): EnqueueStreamRequest;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;
    getPriority(): number;
    setPriority(value: number): EnqueueStreamRequest;
    getHandler(): string;
    setHandler(value: string): EnqueueStreamRequest;

    getParametersMap(): jspb.Map<string, string>;
    clearParametersMap(): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): EnqueueStreamRequest.AsObject;
    static toObject(includeInstance: boolean, msg: EnqueueStreamRequest): EnqueueStreamRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: EnqueueStreamRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): EnqueueStreamRequest;
    static deserializeBinaryFromReader(message: EnqueueStreamRequest, reader: jspb.BinaryReader): EnqueueStreamRequest;
}

export namespace EnqueueStreamRequest {
    export type AsObject = {
        clientmessageid: string,
        tenantcode: string,
        queuecode: string,
        vnamespace: string,
        content: string,
        contenttype: string,

        headersMap: Array<[string, string]>,
        priority: number,
        handler: string,

        parametersMap: Array<[string, string]>,
    }
}

export class EnqueueStreamResponse extends jspb.Message { 
    getClientmessageid(): string;
    setClientmessageid(value: string): EnqueueStreamResponse;
    getConfirmed(): boolean;
    setConfirmed(value: boolean): EnqueueStreamResponse;
    getMessageid(): string;
    setMessageid(value: string): EnqueueStreamResponse;
    getError(): string;
    setError(value: string): EnqueueStreamResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): EnqueueStreamResponse.AsObject;
    static toObject(includeInstance: boolean, msg: EnqueueStreamResponse): EnqueueStreamResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: EnqueueStreamResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): EnqueueStreamResponse;
    static deserializeBinaryFromReader(message: EnqueueStreamResponse, reader: jspb.BinaryReader): EnqueueStreamResponse;
}

export namespace EnqueueStreamResponse {
    export type AsObject = {
        clientmessageid: string,
        confirmed: boolean,
        messageid: string,
        error: string,
    }
}

export enum QueueType {
    STANDARD = 0,
    DELAYED = 1,
    DEAD_LETTER = 2,
}

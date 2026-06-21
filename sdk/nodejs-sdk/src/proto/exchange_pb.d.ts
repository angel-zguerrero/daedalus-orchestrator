// package: exchange
// file: exchange.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

export class CreateExchangeRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): CreateExchangeRequest;
    getCode(): string;
    setCode(value: string): CreateExchangeRequest;
    getName(): string;
    setName(value: string): CreateExchangeRequest;
    getType(): string;
    setType(value: string): CreateExchangeRequest;
    getVnamespace(): string;
    setVnamespace(value: string): CreateExchangeRequest;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateExchangeRequest.AsObject;
    static toObject(includeInstance: boolean, msg: CreateExchangeRequest): CreateExchangeRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateExchangeRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateExchangeRequest;
    static deserializeBinaryFromReader(message: CreateExchangeRequest, reader: jspb.BinaryReader): CreateExchangeRequest;
}

export namespace CreateExchangeRequest {
    export type AsObject = {
        tenantcode: string,
        code: string,
        name: string,
        type: string,
        vnamespace: string,

        headersMap: Array<[string, string]>,
    }
}

export class CreateExchangeResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): CreateExchangeResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): Exchange | undefined;
    setResult(value?: Exchange): CreateExchangeResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateExchangeResponse.AsObject;
    static toObject(includeInstance: boolean, msg: CreateExchangeResponse): CreateExchangeResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateExchangeResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateExchangeResponse;
    static deserializeBinaryFromReader(message: CreateExchangeResponse, reader: jspb.BinaryReader): CreateExchangeResponse;
}

export namespace CreateExchangeResponse {
    export type AsObject = {
        message: string,
        result?: Exchange.AsObject,
    }
}

export class BulkCreateExchangeRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): BulkCreateExchangeRequest;
    clearExchangesList(): void;
    getExchangesList(): Array<CreateExchangeItem>;
    setExchangesList(value: Array<CreateExchangeItem>): BulkCreateExchangeRequest;
    addExchanges(value?: CreateExchangeItem, index?: number): CreateExchangeItem;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): BulkCreateExchangeRequest.AsObject;
    static toObject(includeInstance: boolean, msg: BulkCreateExchangeRequest): BulkCreateExchangeRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: BulkCreateExchangeRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): BulkCreateExchangeRequest;
    static deserializeBinaryFromReader(message: BulkCreateExchangeRequest, reader: jspb.BinaryReader): BulkCreateExchangeRequest;
}

export namespace BulkCreateExchangeRequest {
    export type AsObject = {
        tenantcode: string,
        exchangesList: Array<CreateExchangeItem.AsObject>,
    }
}

export class CreateExchangeItem extends jspb.Message { 
    getCode(): string;
    setCode(value: string): CreateExchangeItem;
    getName(): string;
    setName(value: string): CreateExchangeItem;
    getType(): string;
    setType(value: string): CreateExchangeItem;
    getVnamespace(): string;
    setVnamespace(value: string): CreateExchangeItem;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateExchangeItem.AsObject;
    static toObject(includeInstance: boolean, msg: CreateExchangeItem): CreateExchangeItem.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateExchangeItem, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateExchangeItem;
    static deserializeBinaryFromReader(message: CreateExchangeItem, reader: jspb.BinaryReader): CreateExchangeItem;
}

export namespace CreateExchangeItem {
    export type AsObject = {
        code: string,
        name: string,
        type: string,
        vnamespace: string,

        headersMap: Array<[string, string]>,
    }
}

export class BulkCreateExchangeResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): BulkCreateExchangeResponse;
    clearResultList(): void;
    getResultList(): Array<Exchange>;
    setResultList(value: Array<Exchange>): BulkCreateExchangeResponse;
    addResult(value?: Exchange, index?: number): Exchange;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): BulkCreateExchangeResponse.AsObject;
    static toObject(includeInstance: boolean, msg: BulkCreateExchangeResponse): BulkCreateExchangeResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: BulkCreateExchangeResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): BulkCreateExchangeResponse;
    static deserializeBinaryFromReader(message: BulkCreateExchangeResponse, reader: jspb.BinaryReader): BulkCreateExchangeResponse;
}

export namespace BulkCreateExchangeResponse {
    export type AsObject = {
        message: string,
        resultList: Array<Exchange.AsObject>,
    }
}

export class GetExchangeRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): GetExchangeRequest;
    getCode(): string;
    setCode(value: string): GetExchangeRequest;
    getVnamespace(): string;
    setVnamespace(value: string): GetExchangeRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetExchangeRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetExchangeRequest): GetExchangeRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetExchangeRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetExchangeRequest;
    static deserializeBinaryFromReader(message: GetExchangeRequest, reader: jspb.BinaryReader): GetExchangeRequest;
}

export namespace GetExchangeRequest {
    export type AsObject = {
        tenantcode: string,
        code: string,
        vnamespace: string,
    }
}

export class GetExchangeResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): GetExchangeResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): Exchange | undefined;
    setResult(value?: Exchange): GetExchangeResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetExchangeResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetExchangeResponse): GetExchangeResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetExchangeResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetExchangeResponse;
    static deserializeBinaryFromReader(message: GetExchangeResponse, reader: jspb.BinaryReader): GetExchangeResponse;
}

export namespace GetExchangeResponse {
    export type AsObject = {
        message: string,
        result?: Exchange.AsObject,
    }
}

export class GetExchangesRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): GetExchangesRequest;
    getQ(): string;
    setQ(value: string): GetExchangesRequest;
    getCursor(): string;
    setCursor(value: string): GetExchangesRequest;
    getPagesize(): number;
    setPagesize(value: number): GetExchangesRequest;
    getVnamespace(): string;
    setVnamespace(value: string): GetExchangesRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetExchangesRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetExchangesRequest): GetExchangesRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetExchangesRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetExchangesRequest;
    static deserializeBinaryFromReader(message: GetExchangesRequest, reader: jspb.BinaryReader): GetExchangesRequest;
}

export namespace GetExchangesRequest {
    export type AsObject = {
        tenantcode: string,
        q: string,
        cursor: string,
        pagesize: number,
        vnamespace: string,
    }
}

export class Exchange extends jspb.Message { 
    getId(): string;
    setId(value: string): Exchange;
    getCode(): string;
    setCode(value: string): Exchange;
    getName(): string;
    setName(value: string): Exchange;
    getType(): string;
    setType(value: string): Exchange;
    getVnamespace(): string;
    setVnamespace(value: string): Exchange;
    getCreatedat(): string;
    setCreatedat(value: string): Exchange;
    getUpdatedat(): string;
    setUpdatedat(value: string): Exchange;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Exchange.AsObject;
    static toObject(includeInstance: boolean, msg: Exchange): Exchange.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Exchange, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Exchange;
    static deserializeBinaryFromReader(message: Exchange, reader: jspb.BinaryReader): Exchange;
}

export namespace Exchange {
    export type AsObject = {
        id: string,
        code: string,
        name: string,
        type: string,
        vnamespace: string,
        createdat: string,
        updatedat: string,

        headersMap: Array<[string, string]>,
    }
}

export class ExchangeFindResult extends jspb.Message { 
    clearEntitiesList(): void;
    getEntitiesList(): Array<Exchange>;
    setEntitiesList(value: Array<Exchange>): ExchangeFindResult;
    addEntities(value?: Exchange, index?: number): Exchange;
    getCursor(): string;
    setCursor(value: string): ExchangeFindResult;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ExchangeFindResult.AsObject;
    static toObject(includeInstance: boolean, msg: ExchangeFindResult): ExchangeFindResult.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ExchangeFindResult, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ExchangeFindResult;
    static deserializeBinaryFromReader(message: ExchangeFindResult, reader: jspb.BinaryReader): ExchangeFindResult;
}

export namespace ExchangeFindResult {
    export type AsObject = {
        entitiesList: Array<Exchange.AsObject>,
        cursor: string,
    }
}

export class GetExchangesResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): GetExchangesResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): ExchangeFindResult | undefined;
    setResult(value?: ExchangeFindResult): GetExchangesResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetExchangesResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetExchangesResponse): GetExchangesResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetExchangesResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetExchangesResponse;
    static deserializeBinaryFromReader(message: GetExchangesResponse, reader: jspb.BinaryReader): GetExchangesResponse;
}

export namespace GetExchangesResponse {
    export type AsObject = {
        message: string,
        result?: ExchangeFindResult.AsObject,
    }
}

export class DeleteExchangeRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): DeleteExchangeRequest;
    getCode(): string;
    setCode(value: string): DeleteExchangeRequest;
    getVnamespace(): string;
    setVnamespace(value: string): DeleteExchangeRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteExchangeRequest.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteExchangeRequest): DeleteExchangeRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteExchangeRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteExchangeRequest;
    static deserializeBinaryFromReader(message: DeleteExchangeRequest, reader: jspb.BinaryReader): DeleteExchangeRequest;
}

export namespace DeleteExchangeRequest {
    export type AsObject = {
        tenantcode: string,
        code: string,
        vnamespace: string,
    }
}

export class DeleteExchangeResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): DeleteExchangeResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteExchangeResponse.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteExchangeResponse): DeleteExchangeResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteExchangeResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteExchangeResponse;
    static deserializeBinaryFromReader(message: DeleteExchangeResponse, reader: jspb.BinaryReader): DeleteExchangeResponse;
}

export namespace DeleteExchangeResponse {
    export type AsObject = {
        message: string,
    }
}

export class PublishMessageRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): PublishMessageRequest;
    getExchangecode(): string;
    setExchangecode(value: string): PublishMessageRequest;
    getRoutingkeyorpatternorqueuecode(): string;
    setRoutingkeyorpatternorqueuecode(value: string): PublishMessageRequest;
    getVnamespace(): string;
    setVnamespace(value: string): PublishMessageRequest;

    hasMessage(): boolean;
    clearMessage(): void;
    getMessage(): QueueMessage | undefined;
    setMessage(value?: QueueMessage): PublishMessageRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): PublishMessageRequest.AsObject;
    static toObject(includeInstance: boolean, msg: PublishMessageRequest): PublishMessageRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: PublishMessageRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): PublishMessageRequest;
    static deserializeBinaryFromReader(message: PublishMessageRequest, reader: jspb.BinaryReader): PublishMessageRequest;
}

export namespace PublishMessageRequest {
    export type AsObject = {
        tenantcode: string,
        exchangecode: string,
        routingkeyorpatternorqueuecode: string,
        vnamespace: string,
        message?: QueueMessage.AsObject,
    }
}

export class QueueMessage extends jspb.Message { 
    getMessageid(): string;
    setMessageid(value: string): QueueMessage;
    getHandler(): string;
    setHandler(value: string): QueueMessage;
    getPriority(): number;
    setPriority(value: number): QueueMessage;

    getParametersMap(): jspb.Map<string, string>;
    clearParametersMap(): void;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;
    getContenttype(): string;
    setContenttype(value: string): QueueMessage;
    getContent(): Uint8Array | string;
    getContent_asU8(): Uint8Array;
    getContent_asB64(): string;
    setContent(value: Uint8Array | string): QueueMessage;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): QueueMessage.AsObject;
    static toObject(includeInstance: boolean, msg: QueueMessage): QueueMessage.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: QueueMessage, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): QueueMessage;
    static deserializeBinaryFromReader(message: QueueMessage, reader: jspb.BinaryReader): QueueMessage;
}

export namespace QueueMessage {
    export type AsObject = {
        messageid: string,
        handler: string,
        priority: number,

        parametersMap: Array<[string, string]>,

        headersMap: Array<[string, string]>,
        contenttype: string,
        content: Uint8Array | string,
    }
}

export class PublishMessageResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): PublishMessageResponse;

    getQueuemessagesMap(): jspb.Map<string, string>;
    clearQueuemessagesMap(): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): PublishMessageResponse.AsObject;
    static toObject(includeInstance: boolean, msg: PublishMessageResponse): PublishMessageResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: PublishMessageResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): PublishMessageResponse;
    static deserializeBinaryFromReader(message: PublishMessageResponse, reader: jspb.BinaryReader): PublishMessageResponse;
}

export namespace PublishMessageResponse {
    export type AsObject = {
        message: string,

        queuemessagesMap: Array<[string, string]>,
    }
}

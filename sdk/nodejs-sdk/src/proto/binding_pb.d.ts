// package: binding
// file: binding.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

export class CreateBindingRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): CreateBindingRequest;
    getCode(): string;
    setCode(value: string): CreateBindingRequest;
    getExchangecode(): string;
    setExchangecode(value: string): CreateBindingRequest;
    getQueuecode(): string;
    setQueuecode(value: string): CreateBindingRequest;
    getTargetexchangecode(): string;
    setTargetexchangecode(value: string): CreateBindingRequest;
    getAlternateexchangecode(): string;
    setAlternateexchangecode(value: string): CreateBindingRequest;
    getVnamespace(): string;
    setVnamespace(value: string): CreateBindingRequest;
    getRoutingkey(): string;
    setRoutingkey(value: string): CreateBindingRequest;
    getPattern(): string;
    setPattern(value: string): CreateBindingRequest;
    getXmatch(): string;
    setXmatch(value: string): CreateBindingRequest;
    getBindingtype(): string;
    setBindingtype(value: string): CreateBindingRequest;
    getTargetexchangetype(): string;
    setTargetexchangetype(value: string): CreateBindingRequest;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateBindingRequest.AsObject;
    static toObject(includeInstance: boolean, msg: CreateBindingRequest): CreateBindingRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateBindingRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateBindingRequest;
    static deserializeBinaryFromReader(message: CreateBindingRequest, reader: jspb.BinaryReader): CreateBindingRequest;
}

export namespace CreateBindingRequest {
    export type AsObject = {
        tenantcode: string,
        code: string,
        exchangecode: string,
        queuecode: string,
        targetexchangecode: string,
        alternateexchangecode: string,
        vnamespace: string,
        routingkey: string,
        pattern: string,
        xmatch: string,
        bindingtype: string,
        targetexchangetype: string,

        headersMap: Array<[string, string]>,
    }
}

export class CreateBindingResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): CreateBindingResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): Binding | undefined;
    setResult(value?: Binding): CreateBindingResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateBindingResponse.AsObject;
    static toObject(includeInstance: boolean, msg: CreateBindingResponse): CreateBindingResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateBindingResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateBindingResponse;
    static deserializeBinaryFromReader(message: CreateBindingResponse, reader: jspb.BinaryReader): CreateBindingResponse;
}

export namespace CreateBindingResponse {
    export type AsObject = {
        message: string,
        result?: Binding.AsObject,
    }
}

export class BulkCreateBindingRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): BulkCreateBindingRequest;
    clearBindingsList(): void;
    getBindingsList(): Array<CreateBindingItem>;
    setBindingsList(value: Array<CreateBindingItem>): BulkCreateBindingRequest;
    addBindings(value?: CreateBindingItem, index?: number): CreateBindingItem;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): BulkCreateBindingRequest.AsObject;
    static toObject(includeInstance: boolean, msg: BulkCreateBindingRequest): BulkCreateBindingRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: BulkCreateBindingRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): BulkCreateBindingRequest;
    static deserializeBinaryFromReader(message: BulkCreateBindingRequest, reader: jspb.BinaryReader): BulkCreateBindingRequest;
}

export namespace BulkCreateBindingRequest {
    export type AsObject = {
        tenantcode: string,
        bindingsList: Array<CreateBindingItem.AsObject>,
    }
}

export class CreateBindingItem extends jspb.Message { 
    getCode(): string;
    setCode(value: string): CreateBindingItem;
    getExchangecode(): string;
    setExchangecode(value: string): CreateBindingItem;
    getQueuecode(): string;
    setQueuecode(value: string): CreateBindingItem;
    getTargetexchangecode(): string;
    setTargetexchangecode(value: string): CreateBindingItem;
    getAlternateexchangecode(): string;
    setAlternateexchangecode(value: string): CreateBindingItem;
    getVnamespace(): string;
    setVnamespace(value: string): CreateBindingItem;
    getRoutingkey(): string;
    setRoutingkey(value: string): CreateBindingItem;
    getPattern(): string;
    setPattern(value: string): CreateBindingItem;
    getXmatch(): string;
    setXmatch(value: string): CreateBindingItem;
    getBindingtype(): string;
    setBindingtype(value: string): CreateBindingItem;
    getTargetexchangetype(): string;
    setTargetexchangetype(value: string): CreateBindingItem;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateBindingItem.AsObject;
    static toObject(includeInstance: boolean, msg: CreateBindingItem): CreateBindingItem.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateBindingItem, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateBindingItem;
    static deserializeBinaryFromReader(message: CreateBindingItem, reader: jspb.BinaryReader): CreateBindingItem;
}

export namespace CreateBindingItem {
    export type AsObject = {
        code: string,
        exchangecode: string,
        queuecode: string,
        targetexchangecode: string,
        alternateexchangecode: string,
        vnamespace: string,
        routingkey: string,
        pattern: string,
        xmatch: string,
        bindingtype: string,
        targetexchangetype: string,

        headersMap: Array<[string, string]>,
    }
}

export class BulkCreateBindingResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): BulkCreateBindingResponse;
    clearResultsList(): void;
    getResultsList(): Array<Binding>;
    setResultsList(value: Array<Binding>): BulkCreateBindingResponse;
    addResults(value?: Binding, index?: number): Binding;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): BulkCreateBindingResponse.AsObject;
    static toObject(includeInstance: boolean, msg: BulkCreateBindingResponse): BulkCreateBindingResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: BulkCreateBindingResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): BulkCreateBindingResponse;
    static deserializeBinaryFromReader(message: BulkCreateBindingResponse, reader: jspb.BinaryReader): BulkCreateBindingResponse;
}

export namespace BulkCreateBindingResponse {
    export type AsObject = {
        message: string,
        resultsList: Array<Binding.AsObject>,
    }
}

export class GetBindingRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): GetBindingRequest;
    getExchangecode(): string;
    setExchangecode(value: string): GetBindingRequest;
    getQueuecode(): string;
    setQueuecode(value: string): GetBindingRequest;
    getVnamespace(): string;
    setVnamespace(value: string): GetBindingRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetBindingRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetBindingRequest): GetBindingRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetBindingRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetBindingRequest;
    static deserializeBinaryFromReader(message: GetBindingRequest, reader: jspb.BinaryReader): GetBindingRequest;
}

export namespace GetBindingRequest {
    export type AsObject = {
        tenantcode: string,
        exchangecode: string,
        queuecode: string,
        vnamespace: string,
    }
}

export class GetBindingResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): GetBindingResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): Binding | undefined;
    setResult(value?: Binding): GetBindingResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetBindingResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetBindingResponse): GetBindingResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetBindingResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetBindingResponse;
    static deserializeBinaryFromReader(message: GetBindingResponse, reader: jspb.BinaryReader): GetBindingResponse;
}

export namespace GetBindingResponse {
    export type AsObject = {
        message: string,
        result?: Binding.AsObject,
    }
}

export class GetBindingsRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): GetBindingsRequest;
    getQ(): string;
    setQ(value: string): GetBindingsRequest;
    getCursor(): string;
    setCursor(value: string): GetBindingsRequest;
    getPagesize(): number;
    setPagesize(value: number): GetBindingsRequest;
    getVnamespace(): string;
    setVnamespace(value: string): GetBindingsRequest;
    getIncludeobjects(): boolean;
    setIncludeobjects(value: boolean): GetBindingsRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetBindingsRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetBindingsRequest): GetBindingsRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetBindingsRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetBindingsRequest;
    static deserializeBinaryFromReader(message: GetBindingsRequest, reader: jspb.BinaryReader): GetBindingsRequest;
}

export namespace GetBindingsRequest {
    export type AsObject = {
        tenantcode: string,
        q: string,
        cursor: string,
        pagesize: number,
        vnamespace: string,
        includeobjects: boolean,
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
    }
}

export class Queue extends jspb.Message { 
    getId(): string;
    setId(value: string): Queue;
    getCode(): string;
    setCode(value: string): Queue;
    getName(): string;
    setName(value: string): Queue;
    getVnamespace(): string;
    setVnamespace(value: string): Queue;
    getState(): string;
    setState(value: string): Queue;
    getType(): string;
    setType(value: string): Queue;
    getMessagescount(): number;
    setMessagescount(value: number): Queue;
    getCreatedat(): string;
    setCreatedat(value: string): Queue;
    getUpdatedat(): string;
    setUpdatedat(value: string): Queue;

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
        vnamespace: string,
        state: string,
        type: string,
        messagescount: number,
        createdat: string,
        updatedat: string,
    }
}

export class Binding extends jspb.Message { 
    getId(): string;
    setId(value: string): Binding;
    getCode(): string;
    setCode(value: string): Binding;
    getExchangecode(): string;
    setExchangecode(value: string): Binding;
    getQueuecode(): string;
    setQueuecode(value: string): Binding;
    getTargetexchangecode(): string;
    setTargetexchangecode(value: string): Binding;
    getAlternateexchangecode(): string;
    setAlternateexchangecode(value: string): Binding;
    getVnamespace(): string;
    setVnamespace(value: string): Binding;
    getRoutingkey(): string;
    setRoutingkey(value: string): Binding;
    getPattern(): string;
    setPattern(value: string): Binding;
    getXmatch(): string;
    setXmatch(value: string): Binding;
    getBindingtype(): string;
    setBindingtype(value: string): Binding;
    getTargetexchangetype(): string;
    setTargetexchangetype(value: string): Binding;
    getCreatedat(): string;
    setCreatedat(value: string): Binding;
    getUpdatedat(): string;
    setUpdatedat(value: string): Binding;

    hasExchange(): boolean;
    clearExchange(): void;
    getExchange(): Exchange | undefined;
    setExchange(value?: Exchange): Binding;

    hasQueue(): boolean;
    clearQueue(): void;
    getQueue(): Queue | undefined;
    setQueue(value?: Queue): Binding;

    hasTargetexchange(): boolean;
    clearTargetexchange(): void;
    getTargetexchange(): Exchange | undefined;
    setTargetexchange(value?: Exchange): Binding;

    hasAlternateexchange(): boolean;
    clearAlternateexchange(): void;
    getAlternateexchange(): Exchange | undefined;
    setAlternateexchange(value?: Exchange): Binding;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Binding.AsObject;
    static toObject(includeInstance: boolean, msg: Binding): Binding.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Binding, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Binding;
    static deserializeBinaryFromReader(message: Binding, reader: jspb.BinaryReader): Binding;
}

export namespace Binding {
    export type AsObject = {
        id: string,
        code: string,
        exchangecode: string,
        queuecode: string,
        targetexchangecode: string,
        alternateexchangecode: string,
        vnamespace: string,
        routingkey: string,
        pattern: string,
        xmatch: string,
        bindingtype: string,
        targetexchangetype: string,
        createdat: string,
        updatedat: string,
        exchange?: Exchange.AsObject,
        queue?: Queue.AsObject,
        targetexchange?: Exchange.AsObject,
        alternateexchange?: Exchange.AsObject,

        headersMap: Array<[string, string]>,
    }
}

export class BindingFindResult extends jspb.Message { 
    clearEntitiesList(): void;
    getEntitiesList(): Array<Binding>;
    setEntitiesList(value: Array<Binding>): BindingFindResult;
    addEntities(value?: Binding, index?: number): Binding;
    getCursor(): string;
    setCursor(value: string): BindingFindResult;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): BindingFindResult.AsObject;
    static toObject(includeInstance: boolean, msg: BindingFindResult): BindingFindResult.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: BindingFindResult, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): BindingFindResult;
    static deserializeBinaryFromReader(message: BindingFindResult, reader: jspb.BinaryReader): BindingFindResult;
}

export namespace BindingFindResult {
    export type AsObject = {
        entitiesList: Array<Binding.AsObject>,
        cursor: string,
    }
}

export class GetBindingsResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): GetBindingsResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): BindingFindResult | undefined;
    setResult(value?: BindingFindResult): GetBindingsResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetBindingsResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetBindingsResponse): GetBindingsResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetBindingsResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetBindingsResponse;
    static deserializeBinaryFromReader(message: GetBindingsResponse, reader: jspb.BinaryReader): GetBindingsResponse;
}

export namespace GetBindingsResponse {
    export type AsObject = {
        message: string,
        result?: BindingFindResult.AsObject,
    }
}

export class DeleteBindingRequest extends jspb.Message { 
    getTenantcode(): string;
    setTenantcode(value: string): DeleteBindingRequest;
    getCode(): string;
    setCode(value: string): DeleteBindingRequest;
    getVnamespace(): string;
    setVnamespace(value: string): DeleteBindingRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteBindingRequest.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteBindingRequest): DeleteBindingRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteBindingRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteBindingRequest;
    static deserializeBinaryFromReader(message: DeleteBindingRequest, reader: jspb.BinaryReader): DeleteBindingRequest;
}

export namespace DeleteBindingRequest {
    export type AsObject = {
        tenantcode: string,
        code: string,
        vnamespace: string,
    }
}

export class DeleteBindingResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): DeleteBindingResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteBindingResponse.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteBindingResponse): DeleteBindingResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteBindingResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteBindingResponse;
    static deserializeBinaryFromReader(message: DeleteBindingResponse, reader: jspb.BinaryReader): DeleteBindingResponse;
}

export namespace DeleteBindingResponse {
    export type AsObject = {
        message: string,
    }
}

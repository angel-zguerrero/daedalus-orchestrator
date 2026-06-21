// package: nodescheduler
// file: nodescheduler.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

export class GetNodeSchedulersRequest extends jspb.Message { 
    getQ(): string;
    setQ(value: string): GetNodeSchedulersRequest;
    getCursor(): string;
    setCursor(value: string): GetNodeSchedulersRequest;
    getPagesize(): number;
    setPagesize(value: number): GetNodeSchedulersRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetNodeSchedulersRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetNodeSchedulersRequest): GetNodeSchedulersRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetNodeSchedulersRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetNodeSchedulersRequest;
    static deserializeBinaryFromReader(message: GetNodeSchedulersRequest, reader: jspb.BinaryReader): GetNodeSchedulersRequest;
}

export namespace GetNodeSchedulersRequest {
    export type AsObject = {
        q: string,
        cursor: string,
        pagesize: number,
    }
}

export class NodeScheduler extends jspb.Message { 
    getId(): string;
    setId(value: string): NodeScheduler;
    getName(): string;
    setName(value: string): NodeScheduler;
    getTtl(): number;
    setTtl(value: number): NodeScheduler;
    getLastheartbeat(): string;
    setLastheartbeat(value: string): NodeScheduler;

    getInformationMap(): jspb.Map<string, string>;
    clearInformationMap(): void;
    getConnectionstatus(): string;
    setConnectionstatus(value: string): NodeScheduler;
    getCreatedat(): string;
    setCreatedat(value: string): NodeScheduler;
    getUpdatedat(): string;
    setUpdatedat(value: string): NodeScheduler;
    getAssignedtenantnodeindex(): number;
    setAssignedtenantnodeindex(value: number): NodeScheduler;
    getRunningstatus(): string;
    setRunningstatus(value: string): NodeScheduler;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): NodeScheduler.AsObject;
    static toObject(includeInstance: boolean, msg: NodeScheduler): NodeScheduler.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: NodeScheduler, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): NodeScheduler;
    static deserializeBinaryFromReader(message: NodeScheduler, reader: jspb.BinaryReader): NodeScheduler;
}

export namespace NodeScheduler {
    export type AsObject = {
        id: string,
        name: string,
        ttl: number,
        lastheartbeat: string,

        informationMap: Array<[string, string]>,
        connectionstatus: string,
        createdat: string,
        updatedat: string,
        assignedtenantnodeindex: number,
        runningstatus: string,
    }
}

export class NodeSchedulerFindResult extends jspb.Message { 
    clearEntitiesList(): void;
    getEntitiesList(): Array<NodeScheduler>;
    setEntitiesList(value: Array<NodeScheduler>): NodeSchedulerFindResult;
    addEntities(value?: NodeScheduler, index?: number): NodeScheduler;
    getCursor(): string;
    setCursor(value: string): NodeSchedulerFindResult;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): NodeSchedulerFindResult.AsObject;
    static toObject(includeInstance: boolean, msg: NodeSchedulerFindResult): NodeSchedulerFindResult.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: NodeSchedulerFindResult, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): NodeSchedulerFindResult;
    static deserializeBinaryFromReader(message: NodeSchedulerFindResult, reader: jspb.BinaryReader): NodeSchedulerFindResult;
}

export namespace NodeSchedulerFindResult {
    export type AsObject = {
        entitiesList: Array<NodeScheduler.AsObject>,
        cursor: string,
    }
}

export class GetNodeSchedulersResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): GetNodeSchedulersResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): NodeSchedulerFindResult | undefined;
    setResult(value?: NodeSchedulerFindResult): GetNodeSchedulersResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetNodeSchedulersResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetNodeSchedulersResponse): GetNodeSchedulersResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetNodeSchedulersResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetNodeSchedulersResponse;
    static deserializeBinaryFromReader(message: GetNodeSchedulersResponse, reader: jspb.BinaryReader): GetNodeSchedulersResponse;
}

export namespace GetNodeSchedulersResponse {
    export type AsObject = {
        message: string,
        result?: NodeSchedulerFindResult.AsObject,
    }
}

export class GetNodeSchedulerRequest extends jspb.Message { 
    getId(): string;
    setId(value: string): GetNodeSchedulerRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetNodeSchedulerRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetNodeSchedulerRequest): GetNodeSchedulerRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetNodeSchedulerRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetNodeSchedulerRequest;
    static deserializeBinaryFromReader(message: GetNodeSchedulerRequest, reader: jspb.BinaryReader): GetNodeSchedulerRequest;
}

export namespace GetNodeSchedulerRequest {
    export type AsObject = {
        id: string,
    }
}

export class GetNodeSchedulerResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): GetNodeSchedulerResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): NodeScheduler | undefined;
    setResult(value?: NodeScheduler): GetNodeSchedulerResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetNodeSchedulerResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetNodeSchedulerResponse): GetNodeSchedulerResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetNodeSchedulerResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetNodeSchedulerResponse;
    static deserializeBinaryFromReader(message: GetNodeSchedulerResponse, reader: jspb.BinaryReader): GetNodeSchedulerResponse;
}

export namespace GetNodeSchedulerResponse {
    export type AsObject = {
        message: string,
        result?: NodeScheduler.AsObject,
    }
}

// package: tenant
// file: tenant.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

export class TenantInfoRequest extends jspb.Message { 
    getCode(): string;
    setCode(value: string): TenantInfoRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): TenantInfoRequest.AsObject;
    static toObject(includeInstance: boolean, msg: TenantInfoRequest): TenantInfoRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: TenantInfoRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): TenantInfoRequest;
    static deserializeBinaryFromReader(message: TenantInfoRequest, reader: jspb.BinaryReader): TenantInfoRequest;
}

export namespace TenantInfoRequest {
    export type AsObject = {
        code: string,
    }
}

export class TenantSummaryRequest extends jspb.Message { 
    getCode(): string;
    setCode(value: string): TenantSummaryRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): TenantSummaryRequest.AsObject;
    static toObject(includeInstance: boolean, msg: TenantSummaryRequest): TenantSummaryRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: TenantSummaryRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): TenantSummaryRequest;
    static deserializeBinaryFromReader(message: TenantSummaryRequest, reader: jspb.BinaryReader): TenantSummaryRequest;
}

export namespace TenantSummaryRequest {
    export type AsObject = {
        code: string,
    }
}

export class TenantSummary extends jspb.Message { 
    getId(): string;
    setId(value: string): TenantSummary;
    getTenantid(): string;
    setTenantid(value: string): TenantSummary;
    getCode(): string;
    setCode(value: string): TenantSummary;
    getExchangescount(): number;
    setExchangescount(value: number): TenantSummary;
    getQueuescount(): number;
    setQueuescount(value: number): TenantSummary;
    getMessagescount(): number;
    setMessagescount(value: number): TenantSummary;
    getCreatedat(): string;
    setCreatedat(value: string): TenantSummary;
    getUpdatedat(): string;
    setUpdatedat(value: string): TenantSummary;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): TenantSummary.AsObject;
    static toObject(includeInstance: boolean, msg: TenantSummary): TenantSummary.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: TenantSummary, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): TenantSummary;
    static deserializeBinaryFromReader(message: TenantSummary, reader: jspb.BinaryReader): TenantSummary;
}

export namespace TenantSummary {
    export type AsObject = {
        id: string,
        tenantid: string,
        code: string,
        exchangescount: number,
        queuescount: number,
        messagescount: number,
        createdat: string,
        updatedat: string,
    }
}

export class TenantSummaryResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): TenantSummaryResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): TenantSummary | undefined;
    setResult(value?: TenantSummary): TenantSummaryResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): TenantSummaryResponse.AsObject;
    static toObject(includeInstance: boolean, msg: TenantSummaryResponse): TenantSummaryResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: TenantSummaryResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): TenantSummaryResponse;
    static deserializeBinaryFromReader(message: TenantSummaryResponse, reader: jspb.BinaryReader): TenantSummaryResponse;
}

export namespace TenantSummaryResponse {
    export type AsObject = {
        message: string,
        result?: TenantSummary.AsObject,
    }
}

export class SelfMember extends jspb.Message { 
    getIp(): string;
    setIp(value: string): SelfMember;
    getPort(): number;
    setPort(value: number): SelfMember;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): SelfMember.AsObject;
    static toObject(includeInstance: boolean, msg: SelfMember): SelfMember.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: SelfMember, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): SelfMember;
    static deserializeBinaryFromReader(message: SelfMember, reader: jspb.BinaryReader): SelfMember;
}

export namespace SelfMember {
    export type AsObject = {
        ip: string,
        port: number,
    }
}

export class Node extends jspb.Message { 

    hasSelfmember(): boolean;
    clearSelfmember(): void;
    getSelfmember(): SelfMember | undefined;
    setSelfmember(value?: SelfMember): Node;
    getShardid(): number;
    setShardid(value: number): Node;
    clearRolesList(): void;
    getRolesList(): Array<string>;
    setRolesList(value: Array<string>): Node;
    addRoles(value: string, index?: number): string;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Node.AsObject;
    static toObject(includeInstance: boolean, msg: Node): Node.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Node, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Node;
    static deserializeBinaryFromReader(message: Node, reader: jspb.BinaryReader): Node;
}

export namespace Node {
    export type AsObject = {
        selfmember?: SelfMember.AsObject,
        shardid: number,
        rolesList: Array<string>,
    }
}

export class TenantInfoResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): TenantInfoResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): Tenant | undefined;
    setResult(value?: Tenant): TenantInfoResponse;

    hasNode(): boolean;
    clearNode(): void;
    getNode(): Node | undefined;
    setNode(value?: Node): TenantInfoResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): TenantInfoResponse.AsObject;
    static toObject(includeInstance: boolean, msg: TenantInfoResponse): TenantInfoResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: TenantInfoResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): TenantInfoResponse;
    static deserializeBinaryFromReader(message: TenantInfoResponse, reader: jspb.BinaryReader): TenantInfoResponse;
}

export namespace TenantInfoResponse {
    export type AsObject = {
        message: string,
        result?: Tenant.AsObject,
        node?: Node.AsObject,
    }
}

export class DeleteTenantRequest extends jspb.Message { 
    getCode(): string;
    setCode(value: string): DeleteTenantRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteTenantRequest.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteTenantRequest): DeleteTenantRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteTenantRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteTenantRequest;
    static deserializeBinaryFromReader(message: DeleteTenantRequest, reader: jspb.BinaryReader): DeleteTenantRequest;
}

export namespace DeleteTenantRequest {
    export type AsObject = {
        code: string,
    }
}

export class DeleteTenantResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): DeleteTenantResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteTenantResponse.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteTenantResponse): DeleteTenantResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteTenantResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteTenantResponse;
    static deserializeBinaryFromReader(message: DeleteTenantResponse, reader: jspb.BinaryReader): DeleteTenantResponse;
}

export namespace DeleteTenantResponse {
    export type AsObject = {
        message: string,
    }
}

export class GetTenantsRequest extends jspb.Message { 
    getQ(): string;
    setQ(value: string): GetTenantsRequest;
    getCursor(): string;
    setCursor(value: string): GetTenantsRequest;
    getPagesize(): number;
    setPagesize(value: number): GetTenantsRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetTenantsRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetTenantsRequest): GetTenantsRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetTenantsRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetTenantsRequest;
    static deserializeBinaryFromReader(message: GetTenantsRequest, reader: jspb.BinaryReader): GetTenantsRequest;
}

export namespace GetTenantsRequest {
    export type AsObject = {
        q: string,
        cursor: string,
        pagesize: number,
    }
}

export class Tenant extends jspb.Message { 
    getId(): string;
    setId(value: string): Tenant;
    getName(): string;
    setName(value: string): Tenant;
    getCode(): string;
    setCode(value: string): Tenant;
    getShardid(): number;
    setShardid(value: number): Tenant;
    getStatus(): string;
    setStatus(value: string): Tenant;
    getCreatedat(): string;
    setCreatedat(value: string): Tenant;
    getUpdatedat(): string;
    setUpdatedat(value: string): Tenant;
    getExchangescount(): number;
    setExchangescount(value: number): Tenant;
    getQueuescount(): number;
    setQueuescount(value: number): Tenant;
    getBindingscount(): number;
    setBindingscount(value: number): Tenant;
    getMessagescount(): number;
    setMessagescount(value: number): Tenant;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Tenant.AsObject;
    static toObject(includeInstance: boolean, msg: Tenant): Tenant.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Tenant, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Tenant;
    static deserializeBinaryFromReader(message: Tenant, reader: jspb.BinaryReader): Tenant;
}

export namespace Tenant {
    export type AsObject = {
        id: string,
        name: string,
        code: string,
        shardid: number,
        status: string,
        createdat: string,
        updatedat: string,
        exchangescount: number,
        queuescount: number,
        bindingscount: number,
        messagescount: number,
    }
}

export class FindResult extends jspb.Message { 
    clearEntitiesList(): void;
    getEntitiesList(): Array<Tenant>;
    setEntitiesList(value: Array<Tenant>): FindResult;
    addEntities(value?: Tenant, index?: number): Tenant;
    getCursor(): string;
    setCursor(value: string): FindResult;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): FindResult.AsObject;
    static toObject(includeInstance: boolean, msg: FindResult): FindResult.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: FindResult, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): FindResult;
    static deserializeBinaryFromReader(message: FindResult, reader: jspb.BinaryReader): FindResult;
}

export namespace FindResult {
    export type AsObject = {
        entitiesList: Array<Tenant.AsObject>,
        cursor: string,
    }
}

export class GetTenantsResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): GetTenantsResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): FindResult | undefined;
    setResult(value?: FindResult): GetTenantsResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetTenantsResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetTenantsResponse): GetTenantsResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetTenantsResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetTenantsResponse;
    static deserializeBinaryFromReader(message: GetTenantsResponse, reader: jspb.BinaryReader): GetTenantsResponse;
}

export namespace GetTenantsResponse {
    export type AsObject = {
        message: string,
        result?: FindResult.AsObject,
    }
}

export class AssertTenantRequest extends jspb.Message { 
    getCode(): string;
    setCode(value: string): AssertTenantRequest;
    getName(): string;
    setName(value: string): AssertTenantRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): AssertTenantRequest.AsObject;
    static toObject(includeInstance: boolean, msg: AssertTenantRequest): AssertTenantRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: AssertTenantRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): AssertTenantRequest;
    static deserializeBinaryFromReader(message: AssertTenantRequest, reader: jspb.BinaryReader): AssertTenantRequest;
}

export namespace AssertTenantRequest {
    export type AsObject = {
        code: string,
        name: string,
    }
}

export class AssertBulkTenantRequest extends jspb.Message { 
    clearTenantsList(): void;
    getTenantsList(): Array<AssertTenantRequest>;
    setTenantsList(value: Array<AssertTenantRequest>): AssertBulkTenantRequest;
    addTenants(value?: AssertTenantRequest, index?: number): AssertTenantRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): AssertBulkTenantRequest.AsObject;
    static toObject(includeInstance: boolean, msg: AssertBulkTenantRequest): AssertBulkTenantRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: AssertBulkTenantRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): AssertBulkTenantRequest;
    static deserializeBinaryFromReader(message: AssertBulkTenantRequest, reader: jspb.BinaryReader): AssertBulkTenantRequest;
}

export namespace AssertBulkTenantRequest {
    export type AsObject = {
        tenantsList: Array<AssertTenantRequest.AsObject>,
    }
}

export class AssertBulkTenantResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): AssertBulkTenantResponse;
    clearResultList(): void;
    getResultList(): Array<Tenant>;
    setResultList(value: Array<Tenant>): AssertBulkTenantResponse;
    addResult(value?: Tenant, index?: number): Tenant;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): AssertBulkTenantResponse.AsObject;
    static toObject(includeInstance: boolean, msg: AssertBulkTenantResponse): AssertBulkTenantResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: AssertBulkTenantResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): AssertBulkTenantResponse;
    static deserializeBinaryFromReader(message: AssertBulkTenantResponse, reader: jspb.BinaryReader): AssertBulkTenantResponse;
}

export namespace AssertBulkTenantResponse {
    export type AsObject = {
        message: string,
        resultList: Array<Tenant.AsObject>,
    }
}

export class AssertTenantResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): AssertTenantResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): Tenant | undefined;
    setResult(value?: Tenant): AssertTenantResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): AssertTenantResponse.AsObject;
    static toObject(includeInstance: boolean, msg: AssertTenantResponse): AssertTenantResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: AssertTenantResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): AssertTenantResponse;
    static deserializeBinaryFromReader(message: AssertTenantResponse, reader: jspb.BinaryReader): AssertTenantResponse;
}

export namespace AssertTenantResponse {
    export type AsObject = {
        message: string,
        result?: Tenant.AsObject,
    }
}

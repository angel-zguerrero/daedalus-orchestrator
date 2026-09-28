// package: workflowdefinition
// file: workflow_definition.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

export class WorkflowDefinitionProto extends jspb.Message { 
    getId(): string;
    setId(value: string): WorkflowDefinitionProto;
    getCode(): string;
    setCode(value: string): WorkflowDefinitionProto;
    getVnamespace(): string;
    setVnamespace(value: string): WorkflowDefinitionProto;
    getName(): string;
    setName(value: string): WorkflowDefinitionProto;
    getDescription(): string;
    setDescription(value: string): WorkflowDefinitionProto;
    getVersion(): number;
    setVersion(value: number): WorkflowDefinitionProto;
    getPayload(): Uint8Array | string;
    getPayload_asU8(): Uint8Array;
    getPayload_asB64(): string;
    setPayload(value: Uint8Array | string): WorkflowDefinitionProto;
    getPayloadformat(): string;
    setPayloadformat(value: string): WorkflowDefinitionProto;
    getMaxdurationseconds(): number;
    setMaxdurationseconds(value: number): WorkflowDefinitionProto;
    getIsactive(): boolean;
    setIsactive(value: boolean): WorkflowDefinitionProto;
    getScope(): string;
    setScope(value: string): WorkflowDefinitionProto;
    getTenantid(): string;
    setTenantid(value: string): WorkflowDefinitionProto;
    getCreatedat(): string;
    setCreatedat(value: string): WorkflowDefinitionProto;
    getUpdatedat(): string;
    setUpdatedat(value: string): WorkflowDefinitionProto;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkflowDefinitionProto.AsObject;
    static toObject(includeInstance: boolean, msg: WorkflowDefinitionProto): WorkflowDefinitionProto.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkflowDefinitionProto, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkflowDefinitionProto;
    static deserializeBinaryFromReader(message: WorkflowDefinitionProto, reader: jspb.BinaryReader): WorkflowDefinitionProto;
}

export namespace WorkflowDefinitionProto {
    export type AsObject = {
        id: string,
        code: string,
        vnamespace: string,
        name: string,
        description: string,
        version: number,
        payload: Uint8Array | string,
        payloadformat: string,
        maxdurationseconds: number,
        isactive: boolean,
        scope: string,
        tenantid: string,
        createdat: string,
        updatedat: string,
    }
}

export class CreateWorkflowDefinitionRequest extends jspb.Message { 
    getCode(): string;
    setCode(value: string): CreateWorkflowDefinitionRequest;
    getName(): string;
    setName(value: string): CreateWorkflowDefinitionRequest;
    getDescription(): string;
    setDescription(value: string): CreateWorkflowDefinitionRequest;
    getVersion(): number;
    setVersion(value: number): CreateWorkflowDefinitionRequest;
    getPayload(): Uint8Array | string;
    getPayload_asU8(): Uint8Array;
    getPayload_asB64(): string;
    setPayload(value: Uint8Array | string): CreateWorkflowDefinitionRequest;
    getPayloadformat(): string;
    setPayloadformat(value: string): CreateWorkflowDefinitionRequest;
    getMaxdurationseconds(): number;
    setMaxdurationseconds(value: number): CreateWorkflowDefinitionRequest;
    getIsactive(): boolean;
    setIsactive(value: boolean): CreateWorkflowDefinitionRequest;
    getScope(): string;
    setScope(value: string): CreateWorkflowDefinitionRequest;
    getTenantcode(): string;
    setTenantcode(value: string): CreateWorkflowDefinitionRequest;
    getVnamespace(): string;
    setVnamespace(value: string): CreateWorkflowDefinitionRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateWorkflowDefinitionRequest.AsObject;
    static toObject(includeInstance: boolean, msg: CreateWorkflowDefinitionRequest): CreateWorkflowDefinitionRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateWorkflowDefinitionRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateWorkflowDefinitionRequest;
    static deserializeBinaryFromReader(message: CreateWorkflowDefinitionRequest, reader: jspb.BinaryReader): CreateWorkflowDefinitionRequest;
}

export namespace CreateWorkflowDefinitionRequest {
    export type AsObject = {
        code: string,
        name: string,
        description: string,
        version: number,
        payload: Uint8Array | string,
        payloadformat: string,
        maxdurationseconds: number,
        isactive: boolean,
        scope: string,
        tenantcode: string,
        vnamespace: string,
    }
}

export class CreateWorkflowDefinitionResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): CreateWorkflowDefinitionResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): WorkflowDefinitionProto | undefined;
    setResult(value?: WorkflowDefinitionProto): CreateWorkflowDefinitionResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateWorkflowDefinitionResponse.AsObject;
    static toObject(includeInstance: boolean, msg: CreateWorkflowDefinitionResponse): CreateWorkflowDefinitionResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateWorkflowDefinitionResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateWorkflowDefinitionResponse;
    static deserializeBinaryFromReader(message: CreateWorkflowDefinitionResponse, reader: jspb.BinaryReader): CreateWorkflowDefinitionResponse;
}

export namespace CreateWorkflowDefinitionResponse {
    export type AsObject = {
        message: string,
        result?: WorkflowDefinitionProto.AsObject,
    }
}

export class UpdateWorkflowDefinitionRequest extends jspb.Message { 
    getId(): string;
    setId(value: string): UpdateWorkflowDefinitionRequest;
    getName(): string;
    setName(value: string): UpdateWorkflowDefinitionRequest;
    getDescription(): string;
    setDescription(value: string): UpdateWorkflowDefinitionRequest;
    getVersion(): number;
    setVersion(value: number): UpdateWorkflowDefinitionRequest;
    getPayload(): Uint8Array | string;
    getPayload_asU8(): Uint8Array;
    getPayload_asB64(): string;
    setPayload(value: Uint8Array | string): UpdateWorkflowDefinitionRequest;
    getPayloadformat(): string;
    setPayloadformat(value: string): UpdateWorkflowDefinitionRequest;
    getMaxdurationseconds(): number;
    setMaxdurationseconds(value: number): UpdateWorkflowDefinitionRequest;
    getIsactive(): boolean;
    setIsactive(value: boolean): UpdateWorkflowDefinitionRequest;
    getScope(): string;
    setScope(value: string): UpdateWorkflowDefinitionRequest;
    getTenantcode(): string;
    setTenantcode(value: string): UpdateWorkflowDefinitionRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): UpdateWorkflowDefinitionRequest.AsObject;
    static toObject(includeInstance: boolean, msg: UpdateWorkflowDefinitionRequest): UpdateWorkflowDefinitionRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: UpdateWorkflowDefinitionRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): UpdateWorkflowDefinitionRequest;
    static deserializeBinaryFromReader(message: UpdateWorkflowDefinitionRequest, reader: jspb.BinaryReader): UpdateWorkflowDefinitionRequest;
}

export namespace UpdateWorkflowDefinitionRequest {
    export type AsObject = {
        id: string,
        name: string,
        description: string,
        version: number,
        payload: Uint8Array | string,
        payloadformat: string,
        maxdurationseconds: number,
        isactive: boolean,
        scope: string,
        tenantcode: string,
    }
}

export class UpdateWorkflowDefinitionResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): UpdateWorkflowDefinitionResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): WorkflowDefinitionProto | undefined;
    setResult(value?: WorkflowDefinitionProto): UpdateWorkflowDefinitionResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): UpdateWorkflowDefinitionResponse.AsObject;
    static toObject(includeInstance: boolean, msg: UpdateWorkflowDefinitionResponse): UpdateWorkflowDefinitionResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: UpdateWorkflowDefinitionResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): UpdateWorkflowDefinitionResponse;
    static deserializeBinaryFromReader(message: UpdateWorkflowDefinitionResponse, reader: jspb.BinaryReader): UpdateWorkflowDefinitionResponse;
}

export namespace UpdateWorkflowDefinitionResponse {
    export type AsObject = {
        message: string,
        result?: WorkflowDefinitionProto.AsObject,
    }
}

export class GetWorkflowDefinitionRequest extends jspb.Message { 
    getId(): string;
    setId(value: string): GetWorkflowDefinitionRequest;
    getCode(): string;
    setCode(value: string): GetWorkflowDefinitionRequest;
    getScope(): string;
    setScope(value: string): GetWorkflowDefinitionRequest;
    getTenantcode(): string;
    setTenantcode(value: string): GetWorkflowDefinitionRequest;
    getVnamespace(): string;
    setVnamespace(value: string): GetWorkflowDefinitionRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetWorkflowDefinitionRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetWorkflowDefinitionRequest): GetWorkflowDefinitionRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetWorkflowDefinitionRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetWorkflowDefinitionRequest;
    static deserializeBinaryFromReader(message: GetWorkflowDefinitionRequest, reader: jspb.BinaryReader): GetWorkflowDefinitionRequest;
}

export namespace GetWorkflowDefinitionRequest {
    export type AsObject = {
        id: string,
        code: string,
        scope: string,
        tenantcode: string,
        vnamespace: string,
    }
}

export class GetWorkflowDefinitionResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): GetWorkflowDefinitionResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): WorkflowDefinitionProto | undefined;
    setResult(value?: WorkflowDefinitionProto): GetWorkflowDefinitionResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetWorkflowDefinitionResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetWorkflowDefinitionResponse): GetWorkflowDefinitionResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetWorkflowDefinitionResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetWorkflowDefinitionResponse;
    static deserializeBinaryFromReader(message: GetWorkflowDefinitionResponse, reader: jspb.BinaryReader): GetWorkflowDefinitionResponse;
}

export namespace GetWorkflowDefinitionResponse {
    export type AsObject = {
        message: string,
        result?: WorkflowDefinitionProto.AsObject,
    }
}

export class ListWorkflowDefinitionsRequest extends jspb.Message { 
    getScope(): string;
    setScope(value: string): ListWorkflowDefinitionsRequest;
    getTenantcode(): string;
    setTenantcode(value: string): ListWorkflowDefinitionsRequest;
    getCursor(): string;
    setCursor(value: string): ListWorkflowDefinitionsRequest;
    getPagesize(): number;
    setPagesize(value: number): ListWorkflowDefinitionsRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ListWorkflowDefinitionsRequest.AsObject;
    static toObject(includeInstance: boolean, msg: ListWorkflowDefinitionsRequest): ListWorkflowDefinitionsRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ListWorkflowDefinitionsRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ListWorkflowDefinitionsRequest;
    static deserializeBinaryFromReader(message: ListWorkflowDefinitionsRequest, reader: jspb.BinaryReader): ListWorkflowDefinitionsRequest;
}

export namespace ListWorkflowDefinitionsRequest {
    export type AsObject = {
        scope: string,
        tenantcode: string,
        cursor: string,
        pagesize: number,
    }
}

export class WorkflowDefinitionFindResult extends jspb.Message { 
    clearEntitiesList(): void;
    getEntitiesList(): Array<WorkflowDefinitionProto>;
    setEntitiesList(value: Array<WorkflowDefinitionProto>): WorkflowDefinitionFindResult;
    addEntities(value?: WorkflowDefinitionProto, index?: number): WorkflowDefinitionProto;
    getCursor(): string;
    setCursor(value: string): WorkflowDefinitionFindResult;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): WorkflowDefinitionFindResult.AsObject;
    static toObject(includeInstance: boolean, msg: WorkflowDefinitionFindResult): WorkflowDefinitionFindResult.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: WorkflowDefinitionFindResult, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): WorkflowDefinitionFindResult;
    static deserializeBinaryFromReader(message: WorkflowDefinitionFindResult, reader: jspb.BinaryReader): WorkflowDefinitionFindResult;
}

export namespace WorkflowDefinitionFindResult {
    export type AsObject = {
        entitiesList: Array<WorkflowDefinitionProto.AsObject>,
        cursor: string,
    }
}

export class ListWorkflowDefinitionsResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): ListWorkflowDefinitionsResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): WorkflowDefinitionFindResult | undefined;
    setResult(value?: WorkflowDefinitionFindResult): ListWorkflowDefinitionsResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ListWorkflowDefinitionsResponse.AsObject;
    static toObject(includeInstance: boolean, msg: ListWorkflowDefinitionsResponse): ListWorkflowDefinitionsResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ListWorkflowDefinitionsResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ListWorkflowDefinitionsResponse;
    static deserializeBinaryFromReader(message: ListWorkflowDefinitionsResponse, reader: jspb.BinaryReader): ListWorkflowDefinitionsResponse;
}

export namespace ListWorkflowDefinitionsResponse {
    export type AsObject = {
        message: string,
        result?: WorkflowDefinitionFindResult.AsObject,
    }
}

export class DeleteWorkflowDefinitionRequest extends jspb.Message { 
    getId(): string;
    setId(value: string): DeleteWorkflowDefinitionRequest;
    getScope(): string;
    setScope(value: string): DeleteWorkflowDefinitionRequest;
    getTenantcode(): string;
    setTenantcode(value: string): DeleteWorkflowDefinitionRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteWorkflowDefinitionRequest.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteWorkflowDefinitionRequest): DeleteWorkflowDefinitionRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteWorkflowDefinitionRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteWorkflowDefinitionRequest;
    static deserializeBinaryFromReader(message: DeleteWorkflowDefinitionRequest, reader: jspb.BinaryReader): DeleteWorkflowDefinitionRequest;
}

export namespace DeleteWorkflowDefinitionRequest {
    export type AsObject = {
        id: string,
        scope: string,
        tenantcode: string,
    }
}

export class DeleteWorkflowDefinitionResponse extends jspb.Message { 
    getMessage(): string;
    setMessage(value: string): DeleteWorkflowDefinitionResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteWorkflowDefinitionResponse.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteWorkflowDefinitionResponse): DeleteWorkflowDefinitionResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteWorkflowDefinitionResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteWorkflowDefinitionResponse;
    static deserializeBinaryFromReader(message: DeleteWorkflowDefinitionResponse, reader: jspb.BinaryReader): DeleteWorkflowDefinitionResponse;
}

export namespace DeleteWorkflowDefinitionResponse {
    export type AsObject = {
        message: string,
    }
}

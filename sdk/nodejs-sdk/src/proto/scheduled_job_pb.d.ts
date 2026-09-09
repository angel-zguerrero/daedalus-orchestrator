// package: scheduledjob
// file: scheduled_job.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

export class CreateOneOffScheduledJobRequest extends jspb.Message {
    getTenantcode(): string;
    setTenantcode(value: string): CreateOneOffScheduledJobRequest;
    getTargettype(): string;
    setTargettype(value: string): CreateOneOffScheduledJobRequest;
    getTargetcode(): string;
    setTargetcode(value: string): CreateOneOffScheduledJobRequest;
    getVnamespace(): string;
    setVnamespace(value: string): CreateOneOffScheduledJobRequest;
    getContent(): string;
    setContent(value: string): CreateOneOffScheduledJobRequest;
    getContenttype(): string;
    setContenttype(value: string): CreateOneOffScheduledJobRequest;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;
    getHandler(): string;
    setHandler(value: string): CreateOneOffScheduledJobRequest;

    getParametersMap(): jspb.Map<string, string>;
    clearParametersMap(): void;
    getPriority(): number;
    setPriority(value: number): CreateOneOffScheduledJobRequest;
    getRunat(): string;
    setRunat(value: string): CreateOneOffScheduledJobRequest;
    getRunafter(): string;
    setRunafter(value: string): CreateOneOffScheduledJobRequest;
    getCode(): string;
    setCode(value: string): CreateOneOffScheduledJobRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateOneOffScheduledJobRequest.AsObject;
    static toObject(includeInstance: boolean, msg: CreateOneOffScheduledJobRequest): CreateOneOffScheduledJobRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateOneOffScheduledJobRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateOneOffScheduledJobRequest;
    static deserializeBinaryFromReader(message: CreateOneOffScheduledJobRequest, reader: jspb.BinaryReader): CreateOneOffScheduledJobRequest;
}

export namespace CreateOneOffScheduledJobRequest {
    export type AsObject = {
        tenantcode: string,
        targettype: string,
        targetcode: string,
        vnamespace: string,
        content: string,
        contenttype: string,

        headersMap: Array<[string, string]>,
        handler: string,

        parametersMap: Array<[string, string]>,
        priority: number,
        runat: string,
        runafter: string,
        code: string,
    }
}

export class CreateRecurringScheduledJobRequest extends jspb.Message {
    getTenantcode(): string;
    setTenantcode(value: string): CreateRecurringScheduledJobRequest;
    getTargettype(): string;
    setTargettype(value: string): CreateRecurringScheduledJobRequest;
    getTargetcode(): string;
    setTargetcode(value: string): CreateRecurringScheduledJobRequest;
    getVnamespace(): string;
    setVnamespace(value: string): CreateRecurringScheduledJobRequest;
    getContent(): string;
    setContent(value: string): CreateRecurringScheduledJobRequest;
    getContenttype(): string;
    setContenttype(value: string): CreateRecurringScheduledJobRequest;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;
    getHandler(): string;
    setHandler(value: string): CreateRecurringScheduledJobRequest;

    getParametersMap(): jspb.Map<string, string>;
    clearParametersMap(): void;
    getPriority(): number;
    setPriority(value: number): CreateRecurringScheduledJobRequest;
    getEvery(): string;
    setEvery(value: string): CreateRecurringScheduledJobRequest;
    getCronexpression(): string;
    setCronexpression(value: string): CreateRecurringScheduledJobRequest;
    getCode(): string;
    setCode(value: string): CreateRecurringScheduledJobRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateRecurringScheduledJobRequest.AsObject;
    static toObject(includeInstance: boolean, msg: CreateRecurringScheduledJobRequest): CreateRecurringScheduledJobRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateRecurringScheduledJobRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateRecurringScheduledJobRequest;
    static deserializeBinaryFromReader(message: CreateRecurringScheduledJobRequest, reader: jspb.BinaryReader): CreateRecurringScheduledJobRequest;
}

export namespace CreateRecurringScheduledJobRequest {
    export type AsObject = {
        tenantcode: string,
        targettype: string,
        targetcode: string,
        vnamespace: string,
        content: string,
        contenttype: string,

        headersMap: Array<[string, string]>,
        handler: string,

        parametersMap: Array<[string, string]>,
        priority: number,
        every: string,
        cronexpression: string,
        code: string,
    }
}

export class ScheduledJob extends jspb.Message {
    getId(): string;
    setId(value: string): ScheduledJob;
    getCode(): string;
    setCode(value: string): ScheduledJob;
    getTenantid(): string;
    setTenantid(value: string): ScheduledJob;
    getTargettype(): string;
    setTargettype(value: string): ScheduledJob;
    getTargetid(): string;
    setTargetid(value: string): ScheduledJob;
    getTargetcode(): string;
    setTargetcode(value: string): ScheduledJob;
    getRoutingkeyorpatternorqueuecode(): string;
    setRoutingkeyorpatternorqueuecode(value: string): ScheduledJob;
    getVnamespace(): string;
    setVnamespace(value: string): ScheduledJob;
    getContent(): string;
    setContent(value: string): ScheduledJob;
    getContenttype(): string;
    setContenttype(value: string): ScheduledJob;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;
    getHandler(): string;
    setHandler(value: string): ScheduledJob;

    getParametersMap(): jspb.Map<string, string>;
    clearParametersMap(): void;
    getPriority(): number;
    setPriority(value: number): ScheduledJob;
    getState(): string;
    setState(value: string): ScheduledJob;
    getType(): string;
    setType(value: string): ScheduledJob;
    getEvery(): string;
    setEvery(value: string): ScheduledJob;
    getCronexpression(): string;
    setCronexpression(value: string): ScheduledJob;
    getRunat(): string;
    setRunat(value: string): ScheduledJob;
    getRunafter(): string;
    setRunafter(value: string): ScheduledJob;
    getNextrunat(): string;
    setNextrunat(value: string): ScheduledJob;
    getCreatedat(): string;
    setCreatedat(value: string): ScheduledJob;
    getUpdatedat(): string;
    setUpdatedat(value: string): ScheduledJob;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ScheduledJob.AsObject;
    static toObject(includeInstance: boolean, msg: ScheduledJob): ScheduledJob.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ScheduledJob, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ScheduledJob;
    static deserializeBinaryFromReader(message: ScheduledJob, reader: jspb.BinaryReader): ScheduledJob;
}

export namespace ScheduledJob {
    export type AsObject = {
        id: string,
        code: string,
        tenantid: string,
        targettype: string,
        targetid: string,
        targetcode: string,
        routingkeyorpatternorqueuecode: string,
        vnamespace: string,
        content: string,
        contenttype: string,

        headersMap: Array<[string, string]>,
        handler: string,

        parametersMap: Array<[string, string]>,
        priority: number,
        state: string,
        type: string,
        every: string,
        cronexpression: string,
        runat: string,
        runafter: string,
        nextrunat: string,
        createdat: string,
        updatedat: string,
    }
}

export class CreateScheduledJobResponse extends jspb.Message {
    getMessage(): string;
    setMessage(value: string): CreateScheduledJobResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): ScheduledJob | undefined;
    setResult(value?: ScheduledJob): CreateScheduledJobResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateScheduledJobResponse.AsObject;
    static toObject(includeInstance: boolean, msg: CreateScheduledJobResponse): CreateScheduledJobResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateScheduledJobResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateScheduledJobResponse;
    static deserializeBinaryFromReader(message: CreateScheduledJobResponse, reader: jspb.BinaryReader): CreateScheduledJobResponse;
}

export namespace CreateScheduledJobResponse {
    export type AsObject = {
        message: string,
        result?: ScheduledJob.AsObject,
    }
}

export class GetScheduledJobRequest extends jspb.Message {
    getTenantcode(): string;
    setTenantcode(value: string): GetScheduledJobRequest;
    getId(): string;
    setId(value: string): GetScheduledJobRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetScheduledJobRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetScheduledJobRequest): GetScheduledJobRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetScheduledJobRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetScheduledJobRequest;
    static deserializeBinaryFromReader(message: GetScheduledJobRequest, reader: jspb.BinaryReader): GetScheduledJobRequest;
}

export namespace GetScheduledJobRequest {
    export type AsObject = {
        tenantcode: string,
        id: string,
    }
}

export class GetScheduledJobResponse extends jspb.Message {
    getMessage(): string;
    setMessage(value: string): GetScheduledJobResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): ScheduledJob | undefined;
    setResult(value?: ScheduledJob): GetScheduledJobResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetScheduledJobResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetScheduledJobResponse): GetScheduledJobResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetScheduledJobResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetScheduledJobResponse;
    static deserializeBinaryFromReader(message: GetScheduledJobResponse, reader: jspb.BinaryReader): GetScheduledJobResponse;
}

export namespace GetScheduledJobResponse {
    export type AsObject = {
        message: string,
        result?: ScheduledJob.AsObject,
    }
}

export class ListScheduledJobsRequest extends jspb.Message {
    getTenantcode(): string;
    setTenantcode(value: string): ListScheduledJobsRequest;
    getVnamespace(): string;
    setVnamespace(value: string): ListScheduledJobsRequest;
    getCursor(): string;
    setCursor(value: string): ListScheduledJobsRequest;
    getPagesize(): number;
    setPagesize(value: number): ListScheduledJobsRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ListScheduledJobsRequest.AsObject;
    static toObject(includeInstance: boolean, msg: ListScheduledJobsRequest): ListScheduledJobsRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ListScheduledJobsRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ListScheduledJobsRequest;
    static deserializeBinaryFromReader(message: ListScheduledJobsRequest, reader: jspb.BinaryReader): ListScheduledJobsRequest;
}

export namespace ListScheduledJobsRequest {
    export type AsObject = {
        tenantcode: string,
        vnamespace: string,
        cursor: string,
        pagesize: number,
    }
}

export class ScheduledJobFindResult extends jspb.Message {
    clearEntitiesList(): void;
    getEntitiesList(): Array<ScheduledJob>;
    setEntitiesList(value: Array<ScheduledJob>): ScheduledJobFindResult;
    addEntities(value?: ScheduledJob, index?: number): ScheduledJob;
    getCursor(): string;
    setCursor(value: string): ScheduledJobFindResult;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ScheduledJobFindResult.AsObject;
    static toObject(includeInstance: boolean, msg: ScheduledJobFindResult): ScheduledJobFindResult.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ScheduledJobFindResult, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ScheduledJobFindResult;
    static deserializeBinaryFromReader(message: ScheduledJobFindResult, reader: jspb.BinaryReader): ScheduledJobFindResult;
}

export namespace ScheduledJobFindResult {
    export type AsObject = {
        entitiesList: Array<ScheduledJob.AsObject>,
        cursor: string,
    }
}

export class ListScheduledJobsResponse extends jspb.Message {
    getMessage(): string;
    setMessage(value: string): ListScheduledJobsResponse;

    hasResult(): boolean;
    clearResult(): void;
    getResult(): ScheduledJobFindResult | undefined;
    setResult(value?: ScheduledJobFindResult): ListScheduledJobsResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ListScheduledJobsResponse.AsObject;
    static toObject(includeInstance: boolean, msg: ListScheduledJobsResponse): ListScheduledJobsResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ListScheduledJobsResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ListScheduledJobsResponse;
    static deserializeBinaryFromReader(message: ListScheduledJobsResponse, reader: jspb.BinaryReader): ListScheduledJobsResponse;
}

export namespace ListScheduledJobsResponse {
    export type AsObject = {
        message: string,
        result?: ScheduledJobFindResult.AsObject,
    }
}

export class DeleteScheduledJobRequest extends jspb.Message {
    getTenantcode(): string;
    setTenantcode(value: string): DeleteScheduledJobRequest;
    getId(): string;
    setId(value: string): DeleteScheduledJobRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteScheduledJobRequest.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteScheduledJobRequest): DeleteScheduledJobRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteScheduledJobRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteScheduledJobRequest;
    static deserializeBinaryFromReader(message: DeleteScheduledJobRequest, reader: jspb.BinaryReader): DeleteScheduledJobRequest;
}

export namespace DeleteScheduledJobRequest {
    export type AsObject = {
        tenantcode: string,
        id: string,
    }
}

export class DeleteScheduledJobResponse extends jspb.Message {
    getMessage(): string;
    setMessage(value: string): DeleteScheduledJobResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteScheduledJobResponse.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteScheduledJobResponse): DeleteScheduledJobResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteScheduledJobResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteScheduledJobResponse;
    static deserializeBinaryFromReader(message: DeleteScheduledJobResponse, reader: jspb.BinaryReader): DeleteScheduledJobResponse;
}

export namespace DeleteScheduledJobResponse {
    export type AsObject = {
        message: string,
    }
}

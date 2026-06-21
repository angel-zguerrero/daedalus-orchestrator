// package: metrics
// file: metrics.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

export class SystemMetricsRequest extends jspb.Message { 

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): SystemMetricsRequest.AsObject;
    static toObject(includeInstance: boolean, msg: SystemMetricsRequest): SystemMetricsRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: SystemMetricsRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): SystemMetricsRequest;
    static deserializeBinaryFromReader(message: SystemMetricsRequest, reader: jspb.BinaryReader): SystemMetricsRequest;
}

export namespace SystemMetricsRequest {
    export type AsObject = {
    }
}

export class SystemMetricsResponse extends jspb.Message { 
    getCpuUsagePercent(): number;
    setCpuUsagePercent(value: number): SystemMetricsResponse;
    getMemoryTotalBytes(): number;
    setMemoryTotalBytes(value: number): SystemMetricsResponse;
    getMemoryUsedBytes(): number;
    setMemoryUsedBytes(value: number): SystemMetricsResponse;
    getMemoryFreeBytes(): number;
    setMemoryFreeBytes(value: number): SystemMetricsResponse;
    getUptimeSeconds(): number;
    setUptimeSeconds(value: number): SystemMetricsResponse;
    getHostname(): string;
    setHostname(value: string): SystemMetricsResponse;
    getNodeType(): string;
    setNodeType(value: string): SystemMetricsResponse;
    getTimestamp(): number;
    setTimestamp(value: number): SystemMetricsResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): SystemMetricsResponse.AsObject;
    static toObject(includeInstance: boolean, msg: SystemMetricsResponse): SystemMetricsResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: SystemMetricsResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): SystemMetricsResponse;
    static deserializeBinaryFromReader(message: SystemMetricsResponse, reader: jspb.BinaryReader): SystemMetricsResponse;
}

export namespace SystemMetricsResponse {
    export type AsObject = {
        cpuUsagePercent: number,
        memoryTotalBytes: number,
        memoryUsedBytes: number,
        memoryFreeBytes: number,
        uptimeSeconds: number,
        hostname: string,
        nodeType: string,
        timestamp: number,
    }
}

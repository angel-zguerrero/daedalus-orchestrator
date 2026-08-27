// package: jobworker
// file: jobworker.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

export class ClaimWorkFilter extends jspb.Message { 
    clearTenantcodesList(): void;
    getTenantcodesList(): Array<string>;
    setTenantcodesList(value: Array<string>): ClaimWorkFilter;
    addTenantcodes(value: string, index?: number): string;
    clearExcludetenantcodesList(): void;
    getExcludetenantcodesList(): Array<string>;
    setExcludetenantcodesList(value: Array<string>): ClaimWorkFilter;
    addExcludetenantcodes(value: string, index?: number): string;
    clearTenantpatternsList(): void;
    getTenantpatternsList(): Array<string>;
    setTenantpatternsList(value: Array<string>): ClaimWorkFilter;
    addTenantpatterns(value: string, index?: number): string;
    clearExcludetenantpatternsList(): void;
    getExcludetenantpatternsList(): Array<string>;
    setExcludetenantpatternsList(value: Array<string>): ClaimWorkFilter;
    addExcludetenantpatterns(value: string, index?: number): string;
    clearVnamespacesList(): void;
    getVnamespacesList(): Array<string>;
    setVnamespacesList(value: Array<string>): ClaimWorkFilter;
    addVnamespaces(value: string, index?: number): string;
    clearExcludevnamespacesList(): void;
    getExcludevnamespacesList(): Array<string>;
    setExcludevnamespacesList(value: Array<string>): ClaimWorkFilter;
    addExcludevnamespaces(value: string, index?: number): string;
    clearVnamespacepatternsList(): void;
    getVnamespacepatternsList(): Array<string>;
    setVnamespacepatternsList(value: Array<string>): ClaimWorkFilter;
    addVnamespacepatterns(value: string, index?: number): string;
    clearExcludevnamespacepatternsList(): void;
    getExcludevnamespacepatternsList(): Array<string>;
    setExcludevnamespacepatternsList(value: Array<string>): ClaimWorkFilter;
    addExcludevnamespacepatterns(value: string, index?: number): string;
    clearQueuecodesList(): void;
    getQueuecodesList(): Array<string>;
    setQueuecodesList(value: Array<string>): ClaimWorkFilter;
    addQueuecodes(value: string, index?: number): string;
    clearExcludequeuecodesList(): void;
    getExcludequeuecodesList(): Array<string>;
    setExcludequeuecodesList(value: Array<string>): ClaimWorkFilter;
    addExcludequeuecodes(value: string, index?: number): string;
    clearQueuepatternsList(): void;
    getQueuepatternsList(): Array<string>;
    setQueuepatternsList(value: Array<string>): ClaimWorkFilter;
    addQueuepatterns(value: string, index?: number): string;
    clearExcludequeuepatternsList(): void;
    getExcludequeuepatternsList(): Array<string>;
    setExcludequeuepatternsList(value: Array<string>): ClaimWorkFilter;
    addExcludequeuepatterns(value: string, index?: number): string;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ClaimWorkFilter.AsObject;
    static toObject(includeInstance: boolean, msg: ClaimWorkFilter): ClaimWorkFilter.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ClaimWorkFilter, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ClaimWorkFilter;
    static deserializeBinaryFromReader(message: ClaimWorkFilter, reader: jspb.BinaryReader): ClaimWorkFilter;
}

export namespace ClaimWorkFilter {
    export type AsObject = {
        tenantcodesList: Array<string>,
        excludetenantcodesList: Array<string>,
        tenantpatternsList: Array<string>,
        excludetenantpatternsList: Array<string>,
        vnamespacesList: Array<string>,
        excludevnamespacesList: Array<string>,
        vnamespacepatternsList: Array<string>,
        excludevnamespacepatternsList: Array<string>,
        queuecodesList: Array<string>,
        excludequeuecodesList: Array<string>,
        queuepatternsList: Array<string>,
        excludequeuepatternsList: Array<string>,
    }
}

export class ClaimWorkCapacityPolicy extends jspb.Message { 
    getMaxqueuemessages(): number;
    setMaxqueuemessages(value: number): ClaimWorkCapacityPolicy;
    getCurrentqueuemessages(): number;
    setCurrentqueuemessages(value: number): ClaimWorkCapacityPolicy;

    hasClaimworkfilter(): boolean;
    clearClaimworkfilter(): void;
    getClaimworkfilter(): ClaimWorkFilter | undefined;
    setClaimworkfilter(value?: ClaimWorkFilter): ClaimWorkCapacityPolicy;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ClaimWorkCapacityPolicy.AsObject;
    static toObject(includeInstance: boolean, msg: ClaimWorkCapacityPolicy): ClaimWorkCapacityPolicy.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ClaimWorkCapacityPolicy, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ClaimWorkCapacityPolicy;
    static deserializeBinaryFromReader(message: ClaimWorkCapacityPolicy, reader: jspb.BinaryReader): ClaimWorkCapacityPolicy;
}

export namespace ClaimWorkCapacityPolicy {
    export type AsObject = {
        maxqueuemessages: number,
        currentqueuemessages: number,
        claimworkfilter?: ClaimWorkFilter.AsObject,
    }
}

export class QueueMessage extends jspb.Message { 
    getId(): string;
    setId(value: string): QueueMessage;
    getMessageid(): string;
    setMessageid(value: string): QueueMessage;
    getContent(): string;
    setContent(value: string): QueueMessage;
    getContenttype(): string;
    setContenttype(value: string): QueueMessage;

    getHeadersMap(): jspb.Map<string, string>;
    clearHeadersMap(): void;
    getQueueid(): string;
    setQueueid(value: string): QueueMessage;
    getPriority(): number;
    setPriority(value: number): QueueMessage;
    getHandler(): string;
    setHandler(value: string): QueueMessage;

    getParametersMap(): jspb.Map<string, string>;
    clearParametersMap(): void;
    getVnamespace(): string;
    setVnamespace(value: string): QueueMessage;
    getCreatedat(): string;
    setCreatedat(value: string): QueueMessage;
    getAttempts(): number;
    setAttempts(value: number): QueueMessage;

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
        id: string,
        messageid: string,
        content: string,
        contenttype: string,

        headersMap: Array<[string, string]>,
        queueid: string,
        priority: number,
        handler: string,

        parametersMap: Array<[string, string]>,
        vnamespace: string,
        createdat: string,
        attempts: number,
    }
}

export class ClaimWorkRequest extends jspb.Message { 
    getWorkerid(): string;
    setWorkerid(value: string): ClaimWorkRequest;

    getInformationMap(): jspb.Map<string, string>;
    clearInformationMap(): void;
    clearCapacitypoliciesList(): void;
    getCapacitypoliciesList(): Array<ClaimWorkCapacityPolicy>;
    setCapacitypoliciesList(value: Array<ClaimWorkCapacityPolicy>): ClaimWorkRequest;
    addCapacitypolicies(value?: ClaimWorkCapacityPolicy, index?: number): ClaimWorkCapacityPolicy;
    getWorkername(): string;
    setWorkername(value: string): ClaimWorkRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ClaimWorkRequest.AsObject;
    static toObject(includeInstance: boolean, msg: ClaimWorkRequest): ClaimWorkRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ClaimWorkRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ClaimWorkRequest;
    static deserializeBinaryFromReader(message: ClaimWorkRequest, reader: jspb.BinaryReader): ClaimWorkRequest;
}

export namespace ClaimWorkRequest {
    export type AsObject = {
        workerid: string,

        informationMap: Array<[string, string]>,
        capacitypoliciesList: Array<ClaimWorkCapacityPolicy.AsObject>,
        workername: string,
    }
}

export class ClaimWorkResponse extends jspb.Message { 
    getKnowledge(): string;
    setKnowledge(value: string): ClaimWorkResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ClaimWorkResponse.AsObject;
    static toObject(includeInstance: boolean, msg: ClaimWorkResponse): ClaimWorkResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ClaimWorkResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ClaimWorkResponse;
    static deserializeBinaryFromReader(message: ClaimWorkResponse, reader: jspb.BinaryReader): ClaimWorkResponse;
}

export namespace ClaimWorkResponse {
    export type AsObject = {
        knowledge: string,
    }
}

export class QueueMessageLease extends jspb.Message { 
    getId(): string;
    setId(value: string): QueueMessageLease;
    getQueuemessageid(): string;
    setQueuemessageid(value: string): QueueMessageLease;
    getWorkerid(): string;
    setWorkerid(value: string): QueueMessageLease;
    getLeaseuntil(): string;
    setLeaseuntil(value: string): QueueMessageLease;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): QueueMessageLease.AsObject;
    static toObject(includeInstance: boolean, msg: QueueMessageLease): QueueMessageLease.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: QueueMessageLease, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): QueueMessageLease;
    static deserializeBinaryFromReader(message: QueueMessageLease, reader: jspb.BinaryReader): QueueMessageLease;
}

export namespace QueueMessageLease {
    export type AsObject = {
        id: string,
        queuemessageid: string,
        workerid: string,
        leaseuntil: string,
    }
}

export class ClaimedQueueMessage extends jspb.Message { 

    hasMessage(): boolean;
    clearMessage(): void;
    getMessage(): QueueMessage | undefined;
    setMessage(value?: QueueMessage): ClaimedQueueMessage;

    hasLease(): boolean;
    clearLease(): void;
    getLease(): QueueMessageLease | undefined;
    setLease(value?: QueueMessageLease): ClaimedQueueMessage;
    getTenantcode(): string;
    setTenantcode(value: string): ClaimedQueueMessage;
    getCapacitypolicyindexmatch(): number;
    setCapacitypolicyindexmatch(value: number): ClaimedQueueMessage;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ClaimedQueueMessage.AsObject;
    static toObject(includeInstance: boolean, msg: ClaimedQueueMessage): ClaimedQueueMessage.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ClaimedQueueMessage, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ClaimedQueueMessage;
    static deserializeBinaryFromReader(message: ClaimedQueueMessage, reader: jspb.BinaryReader): ClaimedQueueMessage;
}

export namespace ClaimedQueueMessage {
    export type AsObject = {
        message?: QueueMessage.AsObject,
        lease?: QueueMessageLease.AsObject,
        tenantcode: string,
        capacitypolicyindexmatch: number,
    }
}

export class ClaimWorkStreamMessage extends jspb.Message { 

    hasAck(): boolean;
    clearAck(): void;
    getAck(): ClaimWorkResponse | undefined;
    setAck(value?: ClaimWorkResponse): ClaimWorkStreamMessage;

    hasClaimedmessage(): boolean;
    clearClaimedmessage(): void;
    getClaimedmessage(): ClaimedQueueMessage | undefined;
    setClaimedmessage(value?: ClaimedQueueMessage): ClaimWorkStreamMessage;

    getMessageCase(): ClaimWorkStreamMessage.MessageCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ClaimWorkStreamMessage.AsObject;
    static toObject(includeInstance: boolean, msg: ClaimWorkStreamMessage): ClaimWorkStreamMessage.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ClaimWorkStreamMessage, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ClaimWorkStreamMessage;
    static deserializeBinaryFromReader(message: ClaimWorkStreamMessage, reader: jspb.BinaryReader): ClaimWorkStreamMessage;
}

export namespace ClaimWorkStreamMessage {
    export type AsObject = {
        ack?: ClaimWorkResponse.AsObject,
        claimedmessage?: ClaimedQueueMessage.AsObject,
    }

    export enum MessageCase {
        MESSAGE_NOT_SET = 0,
        ACK = 1,
        CLAIMEDMESSAGE = 2,
    }

}

export class AckMessageRequest extends jspb.Message { 
    getLeaseid(): string;
    setLeaseid(value: string): AckMessageRequest;
    getTenantcode(): string;
    setTenantcode(value: string): AckMessageRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): AckMessageRequest.AsObject;
    static toObject(includeInstance: boolean, msg: AckMessageRequest): AckMessageRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: AckMessageRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): AckMessageRequest;
    static deserializeBinaryFromReader(message: AckMessageRequest, reader: jspb.BinaryReader): AckMessageRequest;
}

export namespace AckMessageRequest {
    export type AsObject = {
        leaseid: string,
        tenantcode: string,
    }
}

export class AckMessageResponse extends jspb.Message { 
    getSuccess(): boolean;
    setSuccess(value: boolean): AckMessageResponse;
    getMessage(): string;
    setMessage(value: string): AckMessageResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): AckMessageResponse.AsObject;
    static toObject(includeInstance: boolean, msg: AckMessageResponse): AckMessageResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: AckMessageResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): AckMessageResponse;
    static deserializeBinaryFromReader(message: AckMessageResponse, reader: jspb.BinaryReader): AckMessageResponse;
}

export namespace AckMessageResponse {
    export type AsObject = {
        success: boolean,
        message: string,
    }
}

export class BulkAckMessageRequest extends jspb.Message { 
    clearLeaseidsList(): void;
    getLeaseidsList(): Array<string>;
    setLeaseidsList(value: Array<string>): BulkAckMessageRequest;
    addLeaseids(value: string, index?: number): string;
    getTenantcode(): string;
    setTenantcode(value: string): BulkAckMessageRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): BulkAckMessageRequest.AsObject;
    static toObject(includeInstance: boolean, msg: BulkAckMessageRequest): BulkAckMessageRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: BulkAckMessageRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): BulkAckMessageRequest;
    static deserializeBinaryFromReader(message: BulkAckMessageRequest, reader: jspb.BinaryReader): BulkAckMessageRequest;
}

export namespace BulkAckMessageRequest {
    export type AsObject = {
        leaseidsList: Array<string>,
        tenantcode: string,
    }
}

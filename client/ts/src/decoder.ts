/**
 * StateSync Binary Decoder for TypeScript/JavaScript
 *
 * Decodes binary state sync messages from the Go statediff package.
 * Compatible with browsers and Node.js.
 */

// Message types (must match Go constants)
export const MsgFullState = 0x01;
export const MsgPatch = 0x02;
export const MsgPatchBatch = 0x03;

// Operation types
export enum Operation {
  None = 0,
  Add = 1,
  Replace = 2,
  Remove = 3,
  Move = 4,
}

// Field types (must match Go FieldType)
export enum FieldType {
  Invalid = 0,
  Int8 = 1,
  Int16 = 2,
  Int32 = 3,
  Int64 = 4,
  Uint8 = 5,
  Uint16 = 6,
  Uint32 = 7,
  Uint64 = 8,
  Float32 = 9,
  Float64 = 10,
  String = 11,
  Bool = 12,
  Bytes = 13,
  Struct = 14,
  Array = 15,
  Map = 16,
  VarInt = 17,
  VarUint = 18,
  Timestamp = 19,
}

// Schema field definition
export interface FieldMeta {
  index: number;
  name: string;
  type: FieldType;
  elemType?: FieldType;
  childSchema?: Schema;
  keyField?: string;
}

// Schema definition
export interface Schema {
  id: number;
  name: string;
  fields: FieldMeta[];
}

// Decoded change
export interface DecodedChange {
  fieldIndex: number;
  fieldName: string;
  op: Operation;
  value?: any;
  arrayChanges?: ArrayChange[];
  mapChanges?: MapChange[];
}

export interface ArrayChange {
  index: number;
  op: Operation;
  oldIndex?: number;
  value?: any;
}

export interface MapChange {
  key: string;
  op: Operation;
  value?: any;
}

// Decoded patch
export interface DecodedPatch {
  schemaId: number;
  schemaName?: string;
  isFullState: boolean;
  changes: DecodedChange[];
}

/**
 * Schema registry for looking up schemas by ID
 */
export class SchemaRegistry {
  private schemas: Map<number, Schema> = new Map();
  private byName: Map<string, Schema> = new Map();

  register(schema: Schema): void {
    this.schemas.set(schema.id, schema);
    this.byName.set(schema.name, schema);
  }

  get(id: number): Schema | undefined {
    return this.schemas.get(id);
  }

  getByName(name: string): Schema | undefined {
    return this.byName.get(name);
  }
}

/**
 * Binary decoder for state sync messages
 */
export class Decoder {
  private buffer: DataView;
  private pos: number = 0;
  private registry: SchemaRegistry;

  constructor(registry: SchemaRegistry) {
    this.registry = registry;
    this.buffer = new DataView(new ArrayBuffer(0));
  }

  /**
   * Decode a binary message
   */
  decode(data: ArrayBuffer | Uint8Array): DecodedPatch {
    if (data instanceof Uint8Array) {
      this.buffer = new DataView(data.buffer, data.byteOffset, data.byteLength);
    } else {
      this.buffer = new DataView(data);
    }
    this.pos = 0;

    const msgType = this.readByte();

    switch (msgType) {
      case MsgFullState:
        return this.decodeFullState();
      case MsgPatch:
        return this.decodePatch();
      default:
        throw new Error(`Invalid message type: ${msgType}`);
    }
  }

  private decodeFullState(): DecodedPatch {
    const schemaId = this.readUint16();
    const schema = this.registry.get(schemaId);
    if (!schema) {
      throw new Error(`Unknown schema ID: ${schemaId}`);
    }

    const fieldCount = this.readByte();
    const changes: DecodedChange[] = [];

    for (let i = 0; i < fieldCount; i++) {
      const field = schema.fields[i];
      if (!field) continue;

      const value = this.decodeField(field);
      changes.push({
        fieldIndex: i,
        fieldName: field.name,
        op: Operation.Replace,
        value,
      });
    }

    return {
      schemaId,
      schemaName: schema.name,
      isFullState: true,
      changes,
    };
  }

  private decodePatch(): DecodedPatch {
    const schemaId = this.readUint16();
    const schema = this.registry.get(schemaId);
    if (!schema) {
      throw new Error(`Unknown schema ID: ${schemaId}`);
    }

    const changeCount = this.readVarUint();
    const changes: DecodedChange[] = [];

    for (let i = 0; i < changeCount; i++) {
      const fieldIndex = this.readByte();
      const field = schema.fields[fieldIndex];
      if (!field) continue;

      const change: DecodedChange = {
        fieldIndex,
        fieldName: field.name,
        op: Operation.None,
      };

      // Check for array/map incremental changes
      if (field.type === FieldType.Array) {
        const arrayChanges = this.decodeArrayChanges(field);
        if (arrayChanges) {
          change.arrayChanges = arrayChanges;
          changes.push(change);
          continue;
        }
      }

      if (field.type === FieldType.Map) {
        const mapChanges = this.decodeMapChanges(field);
        if (mapChanges) {
          change.mapChanges = mapChanges;
          changes.push(change);
          continue;
        }
      }

      // Simple field change
      const op = this.readByte() as Operation;
      change.op = op;

      if (op !== Operation.Remove) {
        change.value = this.decodeField(field);
      }

      changes.push(change);
    }

    return {
      schemaId,
      schemaName: schema.name,
      isFullState: false,
      changes,
    };
  }

  private decodeField(field: FieldMeta): any {
    switch (field.type) {
      case FieldType.Int8:
        return this.readInt8();
      case FieldType.Int16:
        return this.readInt16();
      case FieldType.Int32:
        return this.readInt32();
      case FieldType.Int64:
        return this.readInt64();
      case FieldType.Uint8:
        return this.readByte();
      case FieldType.Uint16:
        return this.readUint16();
      case FieldType.Uint32:
        return this.readUint32();
      case FieldType.Uint64:
        return this.readUint64();
      case FieldType.Float32:
        return this.readFloat32();
      case FieldType.Float64:
        return this.readFloat64();
      case FieldType.String:
        return this.readString();
      case FieldType.Bool:
        return this.readBool();
      case FieldType.Bytes:
        return this.readBytes();
      case FieldType.VarInt:
        return this.readVarInt();
      case FieldType.VarUint:
        return this.readVarUint();
      case FieldType.Struct:
        return this.decodeStruct(field.childSchema!);
      case FieldType.Array:
        return this.decodeArray(field);
      case FieldType.Map:
        return this.decodeMap(field);
      default:
        throw new Error(`Unknown field type: ${field.type}`);
    }
  }

  private decodeStruct(schema: Schema): Record<string, any> | null {
    const isNull = this.readByte();
    if (isNull === 0) return null;

    const result: Record<string, any> = {};
    for (const field of schema.fields) {
      result[field.name] = this.decodeField(field);
    }
    return result;
  }

  private decodeArray(field: FieldMeta): any[] {
    const op = this.readByte();
    if (op !== Operation.Replace) {
      throw new Error(`Expected Replace op for array, got ${op}`);
    }

    const length = this.readVarUint();
    const result: any[] = [];

    for (let i = 0; i < length; i++) {
      result.push(this.decodeArrayElement(field));
    }

    return result;
  }

  private decodeArrayChanges(field: FieldMeta): ArrayChange[] | null {
    const changeCount = this.readVarUint();
    const changes: ArrayChange[] = [];

    for (let i = 0; i < changeCount; i++) {
      const index = this.readVarUint();
      const op = this.readByte() as Operation;

      const change: ArrayChange = { index, op };

      switch (op) {
        case Operation.Add:
        case Operation.Replace:
          change.value = this.decodeArrayElement(field);
          break;
        case Operation.Move:
          change.oldIndex = this.readVarUint();
          break;
      }

      changes.push(change);
    }

    return changes.length > 0 ? changes : null;
  }

  private decodeArrayElement(field: FieldMeta): any {
    if (field.elemType === FieldType.Struct) {
      return this.decodeStruct(field.childSchema!);
    }
    const tempField: FieldMeta = { index: 0, name: '', type: field.elemType! };
    return this.decodeField(tempField);
  }

  private decodeMap(field: FieldMeta): Record<string, any> {
    const op = this.readByte();
    if (op !== Operation.Replace) {
      throw new Error(`Expected Replace op for map, got ${op}`);
    }

    const length = this.readVarUint();
    const result: Record<string, any> = {};

    for (let i = 0; i < length; i++) {
      const key = this.readString();
      result[key] = this.decodeMapValue(field);
    }

    return result;
  }

  private decodeMapChanges(field: FieldMeta): MapChange[] | null {
    const changeCount = this.readVarUint();
    const changes: MapChange[] = [];

    for (let i = 0; i < changeCount; i++) {
      const key = this.readString();
      const op = this.readByte() as Operation;

      const change: MapChange = { key, op };

      if (op !== Operation.Remove) {
        change.value = this.decodeMapValue(field);
      }

      changes.push(change);
    }

    return changes.length > 0 ? changes : null;
  }

  private decodeMapValue(field: FieldMeta): any {
    if (field.elemType === FieldType.Struct) {
      return this.decodeStruct(field.childSchema!);
    }
    const tempField: FieldMeta = { index: 0, name: '', type: field.elemType! };
    return this.decodeField(tempField);
  }

  // Primitive read methods

  private readByte(): number {
    if (this.pos >= this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    return this.buffer.getUint8(this.pos++);
  }

  private readInt8(): number {
    if (this.pos >= this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    return this.buffer.getInt8(this.pos++);
  }

  private readInt16(): number {
    if (this.pos + 2 > this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    const v = this.buffer.getInt16(this.pos, true); // little-endian
    this.pos += 2;
    return v;
  }

  private readUint16(): number {
    if (this.pos + 2 > this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    const v = this.buffer.getUint16(this.pos, true);
    this.pos += 2;
    return v;
  }

  private readInt32(): number {
    if (this.pos + 4 > this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    const v = this.buffer.getInt32(this.pos, true);
    this.pos += 4;
    return v;
  }

  private readUint32(): number {
    if (this.pos + 4 > this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    const v = this.buffer.getUint32(this.pos, true);
    this.pos += 4;
    return v;
  }

  private readInt64(): bigint {
    if (this.pos + 8 > this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    const v = this.buffer.getBigInt64(this.pos, true);
    this.pos += 8;
    return v;
  }

  private readUint64(): bigint {
    if (this.pos + 8 > this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    const v = this.buffer.getBigUint64(this.pos, true);
    this.pos += 8;
    return v;
  }

  private readFloat32(): number {
    if (this.pos + 4 > this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    const v = this.buffer.getFloat32(this.pos, true);
    this.pos += 4;
    return v;
  }

  private readFloat64(): number {
    if (this.pos + 8 > this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    const v = this.buffer.getFloat64(this.pos, true);
    this.pos += 8;
    return v;
  }

  private readBool(): boolean {
    return this.readByte() !== 0;
  }

  private readString(): string {
    const length = this.readVarUint();
    if (this.pos + length > this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    const bytes = new Uint8Array(this.buffer.buffer, this.buffer.byteOffset + this.pos, length);
    this.pos += length;
    return new TextDecoder().decode(bytes);
  }

  private readBytes(): Uint8Array {
    const length = this.readVarUint();
    if (this.pos + length > this.buffer.byteLength) {
      throw new Error('Buffer underflow');
    }
    const bytes = new Uint8Array(length);
    bytes.set(new Uint8Array(this.buffer.buffer, this.buffer.byteOffset + this.pos, length));
    this.pos += length;
    return bytes;
  }

  private readVarInt(): number {
    const uv = this.readVarUint();
    // Zigzag decode
    return Number((BigInt(uv) >> 1n) ^ -(BigInt(uv) & 1n));
  }

  private readVarUint(): number {
    let result = 0;
    let shift = 0;
    while (true) {
      if (this.pos >= this.buffer.byteLength) {
        throw new Error('Buffer underflow');
      }
      const b = this.buffer.getUint8(this.pos++);
      result |= (b & 0x7f) << shift;
      if ((b & 0x80) === 0) break;
      shift += 7;
      if (shift >= 32) {
        throw new Error('VarUint overflow');
      }
    }
    return result >>> 0; // Convert to unsigned
  }
}

/**
 * State container that applies patches
 */
export class SyncState<T extends Record<string, any>> {
  private state: T;
  private schema: Schema;
  private decoder: Decoder;
  private listeners: Set<(state: T, changes: DecodedChange[]) => void> = new Set();

  constructor(schema: Schema, registry: SchemaRegistry, initialState?: T) {
    this.schema = schema;
    this.decoder = new Decoder(registry);
    this.state = initialState || ({} as T);
  }

  /**
   * Get current state
   */
  get(): T {
    return this.state;
  }

  /**
   * Apply a binary patch/full state message
   */
  apply(data: ArrayBuffer | Uint8Array): DecodedChange[] {
    const patch = this.decoder.decode(data);

    if (patch.isFullState) {
      // Full state replace
      const newState = {} as T;
      for (const change of patch.changes) {
        (newState as any)[change.fieldName] = change.value;
      }
      this.state = newState;
    } else {
      // Apply incremental changes
      for (const change of patch.changes) {
        this.applyChange(change);
      }
    }

    // Notify listeners
    this.listeners.forEach((fn) => fn(this.state, patch.changes));

    return patch.changes;
  }

  private applyChange(change: DecodedChange): void {
    const state = this.state as any;

    // Handle array changes
    if (change.arrayChanges) {
      let arr = state[change.fieldName] as any[];
      if (!arr) arr = state[change.fieldName] = [];

      for (const ac of change.arrayChanges) {
        switch (ac.op) {
          case Operation.Add:
            if (ac.index >= arr.length) {
              arr.push(ac.value);
            } else {
              arr.splice(ac.index, 0, ac.value);
            }
            break;
          case Operation.Replace:
            arr[ac.index] = ac.value;
            break;
          case Operation.Remove:
            arr.splice(ac.index, 1);
            break;
          case Operation.Move:
            if (ac.oldIndex !== undefined) {
              const elem = arr.splice(ac.oldIndex, 1)[0];
              arr.splice(ac.index, 0, elem);
            }
            break;
        }
      }
      return;
    }

    // Handle map changes
    if (change.mapChanges) {
      let map = state[change.fieldName] as Record<string, any>;
      if (!map) map = state[change.fieldName] = {};

      for (const mc of change.mapChanges) {
        switch (mc.op) {
          case Operation.Add:
          case Operation.Replace:
            map[mc.key] = mc.value;
            break;
          case Operation.Remove:
            delete map[mc.key];
            break;
        }
      }
      return;
    }

    // Simple field change
    switch (change.op) {
      case Operation.Add:
      case Operation.Replace:
        state[change.fieldName] = change.value;
        break;
      case Operation.Remove:
        delete state[change.fieldName];
        break;
    }
  }

  /**
   * Subscribe to state changes
   */
  onChange(fn: (state: T, changes: DecodedChange[]) => void): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }
}

/**
 * Helper to create a schema from a simple definition
 */
export function defineSchema(
  id: number,
  name: string,
  fields: Array<{ name: string; type: FieldType; elemType?: FieldType; childSchema?: Schema; keyField?: string }>
): Schema {
  return {
    id,
    name,
    fields: fields.map((f, i) => ({
      index: i,
      name: f.name,
      type: f.type,
      elemType: f.elemType,
      childSchema: f.childSchema,
      keyField: f.keyField,
    })),
  };
}

// Export for CommonJS compatibility
export default {
  Decoder,
  SchemaRegistry,
  SyncState,
  defineSchema,
  FieldType,
  Operation,
  MsgFullState,
  MsgPatch,
};

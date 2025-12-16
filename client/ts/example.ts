/**
 * Example usage of the statediff TypeScript client
 */
import { Decoder, SchemaRegistry, SyncState, defineSchema, FieldType } from './src';

// Define schemas (must match Go server schemas)
const playerSchema = defineSchema(2, 'Player', [
  { name: 'ID', type: FieldType.String },
  { name: 'Name', type: FieldType.String },
  { name: 'Score', type: FieldType.Int64 },
]);

const gameSchema = defineSchema(1, 'GameState', [
  { name: 'round', type: FieldType.Int64 },
  { name: 'phase', type: FieldType.String },
  { name: 'players', type: FieldType.Array, elemType: FieldType.Struct, childSchema: playerSchema },
  { name: 'scores', type: FieldType.Map, elemType: FieldType.Int64 },
]);

// Setup registry
const registry = new SchemaRegistry();
registry.register(playerSchema);
registry.register(gameSchema);

// Create state container
interface GameState {
  round: number;
  phase: string;
  players: Array<{ ID: string; Name: string; Score: number }>;
  scores: Record<string, number>;
}

const state = new SyncState<GameState>(gameSchema, registry, {
  round: 0,
  phase: '',
  players: [],
  scores: {},
});

// Subscribe to changes
const unsubscribe = state.onChange((newState, changes) => {
  console.log('State updated:', newState);
  console.log('Changes:', changes);
});

// Example: WebSocket integration
/*
const ws = new WebSocket('ws://localhost:8080/game');
ws.binaryType = 'arraybuffer';

ws.onmessage = (event) => {
  const data = event.data as ArrayBuffer;
  const changes = state.apply(data);

  // React to specific changes
  for (const change of changes) {
    if (change.fieldName === 'phase') {
      console.log('Phase changed to:', state.get().phase);
    }
  }
};
*/

// Example: Manual decoding
const decoder = new Decoder(registry);

// Simulate receiving a binary message (this would come from WebSocket)
// const patch = decoder.decode(binaryData);
// console.log('Decoded patch:', patch);

export { registry, state, decoder };

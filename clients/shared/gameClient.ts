export type Player = {
  id: string;
  name: string;
  x: number;
  y: number;
  score: number;
  ready: boolean;
};

export type ServerEvent = {
  type: "connected" | "player_joined" | "player_left" | "snapshot" | "match_finished" | "error";
  data?: unknown;
};

export class GameClient {
  private socket?: WebSocket;
  private sequence = 0;

  connect(baseUrl: string, roomCode: string, playerId: string, name: string, onEvent: (event: ServerEvent) => void) {
    const query = new URLSearchParams({ playerId, name });
    this.socket = new WebSocket(`${baseUrl}/api/v1/rooms/${roomCode}/connect?${query}`);
    this.socket.onmessage = message => onEvent(JSON.parse(message.data as string));
  }

  ready(value = true) { this.send({ type: "ready", ready: value }); }
  start() { this.send({ type: "start" }); }
  move(dx: number, dy: number) { this.send({ type: "input", dx, dy, sequence: ++this.sequence }); }
  close() { this.socket?.close(); }

  private send(message: object) {
    if (this.socket?.readyState !== WebSocket.OPEN) throw new Error("game socket is not connected");
    this.socket.send(JSON.stringify(message));
  }
}

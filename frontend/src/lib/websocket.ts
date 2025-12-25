import type { Event, ServerMessage, AutocompleteRequest, AutocompleteResponse } from './types';

export enum ConnectionState {
  CONNECTING = 'CONNECTING',
  CONNECTED = 'CONNECTED',
  RECONNECTING = 'RECONNECTING',
  DISCONNECTED = 'DISCONNECTED',
}

export interface WebSocketOptions {
  reconnectDelay?: number;
  maxReconnectAttempts?: number;
}

type MessageHandler = (message: ServerMessage) => void;
type ConnectionHandler = (state: ConnectionState) => void;
type AutocompleteHandler = (response: AutocompleteResponse) => void;

export class TodoWebSocket {
  private url: string;
  private ws: WebSocket | null = null;
  private connectionState: ConnectionState = ConnectionState.CONNECTING;
  private messageHandlers: MessageHandler[] = [];
  private connectionHandlers: ConnectionHandler[] = [];
  private autocompleteHandlers: AutocompleteHandler[] = [];
  private messageQueue: Event[] = [];
  private reconnectAttempts = 0;
  private manualClose = false;
  private options: Required<WebSocketOptions>;

  constructor(url: string, options: WebSocketOptions = {}) {
    this.url = url;
    this.options = {
      reconnectDelay: options.reconnectDelay ?? 1000,
      maxReconnectAttempts: options.maxReconnectAttempts ?? 10,
    };
    this.connect();
  }

  private connect() {
    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      this.reconnectAttempts = 0;
      this.setConnectionState(ConnectionState.CONNECTED);
      this.flushMessageQueue();
    };

    this.ws.onclose = () => {
      if (this.manualClose) {
        this.setConnectionState(ConnectionState.DISCONNECTED);
        return;
      }

      this.setConnectionState(ConnectionState.RECONNECTING);
      this.scheduleReconnect();
    };

    this.ws.onerror = () => {
      // Error handling - close will be called after error
    };

    this.ws.onmessage = (event: MessageEvent) => {
      try {
        const message: ServerMessage = JSON.parse(event.data);
        // Handle autocomplete responses separately
        if (message.type === 'AutocompleteResponse') {
          this.notifyAutocompleteHandlers(message as AutocompleteResponse);
        } else {
          this.notifyMessageHandlers(message);
        }
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e);
      }
    };
  }

  private scheduleReconnect() {
    if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
      this.setConnectionState(ConnectionState.DISCONNECTED);
      return;
    }

    this.reconnectAttempts++;
    setTimeout(() => {
      if (!this.manualClose) {
        this.connect();
      }
    }, this.options.reconnectDelay);
  }

  private flushMessageQueue() {
    while (this.messageQueue.length > 0) {
      const event = this.messageQueue.shift()!;
      this.sendImmediate(event);
    }
  }

  private sendImmediate(event: Event) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(event));
    }
  }

  private setConnectionState(state: ConnectionState) {
    this.connectionState = state;
    this.notifyConnectionHandlers(state);
  }

  private notifyMessageHandlers(message: ServerMessage) {
    for (const handler of this.messageHandlers) {
      handler(message);
    }
  }

  private notifyConnectionHandlers(state: ConnectionState) {
    for (const handler of this.connectionHandlers) {
      handler(state);
    }
  }

  private notifyAutocompleteHandlers(response: AutocompleteResponse) {
    for (const handler of this.autocompleteHandlers) {
      handler(response);
    }
  }

  // Public API

  send(event: Event) {
    if (this.connectionState === ConnectionState.CONNECTED) {
      this.sendImmediate(event);
    } else {
      this.messageQueue.push(event);
    }
  }

  onMessage(handler: MessageHandler) {
    this.messageHandlers.push(handler);
    return () => {
      const index = this.messageHandlers.indexOf(handler);
      if (index > -1) {
        this.messageHandlers.splice(index, 1);
      }
    };
  }

  onConnectionChange(handler: ConnectionHandler) {
    this.connectionHandlers.push(handler);
    return () => {
      const index = this.connectionHandlers.indexOf(handler);
      if (index > -1) {
        this.connectionHandlers.splice(index, 1);
      }
    };
  }

  onAutocomplete(handler: AutocompleteHandler) {
    this.autocompleteHandlers.push(handler);
    return () => {
      const index = this.autocompleteHandlers.indexOf(handler);
      if (index > -1) {
        this.autocompleteHandlers.splice(index, 1);
      }
    };
  }

  sendAutocompleteRequest(query: string, requestId: string) {
    const request: AutocompleteRequest = {
      type: 'AutocompleteRequest',
      query,
      requestId,
    };
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(request));
    }
  }

  getConnectionState(): ConnectionState {
    return this.connectionState;
  }

  close() {
    this.manualClose = true;
    if (this.ws) {
      this.ws.close();
    }
    this.setConnectionState(ConnectionState.DISCONNECTED);
  }
}


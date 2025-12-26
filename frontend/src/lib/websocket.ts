import type { Event, ServerMessage, AutocompleteRequest, AutocompleteResponse, Command } from './types';

export enum ConnectionState {
  CONNECTING = 'CONNECTING',
  CONNECTED = 'CONNECTED',
  RECONNECTING = 'RECONNECTING',
  DISCONNECTED = 'DISCONNECTED',
}

export interface WebSocketOptions {
  reconnectDelay?: number;
  maxReconnectAttempts?: number;
  enableHeartbeat?: boolean; // For testing
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
  private messageQueue: Command[] = [];
  private reconnectAttempts = 0;
  private manualClose = false;
  private reconnectTimeout: number | null = null;
  private options: Required<WebSocketOptions>;
  private visibilityHandler: (() => void) | null = null;
  private onlineHandler: (() => void) | null = null;
  private offlineHandler: (() => void) | null = null;
  private heartbeatInterval: number | null = null;
  private lastHeartbeat: number = Date.now();
  private enableHeartbeat: boolean;

  constructor(url: string, options: WebSocketOptions = {}) {
    this.url = url;
    this.options = {
      reconnectDelay: options.reconnectDelay ?? 1000,
      maxReconnectAttempts: options.maxReconnectAttempts ?? Infinity, // Keep trying indefinitely
      enableHeartbeat: options.enableHeartbeat ?? true,
    };
    this.enableHeartbeat = this.options.enableHeartbeat;
    
    // Set up event listeners for mobile scenarios
    this.setupEventListeners();
    
    this.connect();
  }

  private setupEventListeners() {
    // Handle page visibility changes (app backgrounding on mobile)
    this.visibilityHandler = () => {
      if (document.visibilityState === 'visible') {
        console.log('Page became visible, checking connection...');
        
        // On mobile, WebSocket connections often die when app is backgrounded
        // Check the actual WebSocket state, not just our internal state
        const wsActuallyConnected = this.ws && this.ws.readyState === WebSocket.OPEN;
        
        if (!wsActuallyConnected) {
          // WebSocket is not actually open, force reconnect
          console.log('WebSocket not actually open (readyState:', this.ws?.readyState, '), forcing reconnect...');
          this.reconnect();
        } else if (this.connectionState === ConnectionState.CONNECTED) {
          // We think we're connected, but check if connection is stale
          const timeSinceLastHeartbeat = Date.now() - this.lastHeartbeat;
          if (timeSinceLastHeartbeat > 5000) {
            console.log('Connection may be stale (no messages for', timeSinceLastHeartbeat, 'ms), reconnecting...');
            this.reconnect();
          } else {
            console.log('Connection appears healthy (last message', timeSinceLastHeartbeat, 'ms ago)');
          }
        } else if (this.connectionState === ConnectionState.RECONNECTING || this.connectionState === ConnectionState.CONNECTING) {
          // If we're reconnecting or connecting, try immediately when app comes back
          console.log('App returned while', this.connectionState, ', trying immediately...');
          if (this.reconnectTimeout !== null) {
            clearTimeout(this.reconnectTimeout);
            this.reconnectTimeout = null;
          }
          this.connect();
        }
      } else {
        // Page became hidden (backgrounded)
        console.log('Page became hidden');
      }
    };
    document.addEventListener('visibilitychange', this.visibilityHandler);

    // Handle network status changes
    this.onlineHandler = () => {
      console.log('Network came online, reconnecting...');
      if (this.connectionState !== ConnectionState.CONNECTED && !this.manualClose) {
        // Network came back, try reconnecting immediately
        if (this.reconnectTimeout !== null) {
          clearTimeout(this.reconnectTimeout);
          this.reconnectTimeout = null;
        }
        this.reconnect();
      }
    };
    window.addEventListener('online', this.onlineHandler);

    this.offlineHandler = () => {
      console.log('Network went offline');
      // Don't close connection immediately - let normal error handling do it
    };
    window.addEventListener('offline', this.offlineHandler);
  }

  private cleanupEventListeners() {
    if (this.visibilityHandler) {
      document.removeEventListener('visibilitychange', this.visibilityHandler);
      this.visibilityHandler = null;
    }
    if (this.onlineHandler) {
      window.removeEventListener('online', this.onlineHandler);
      this.onlineHandler = null;
    }
    if (this.offlineHandler) {
      window.removeEventListener('offline', this.offlineHandler);
      this.offlineHandler = null;
    }
    if (this.heartbeatInterval !== null) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private startHeartbeat() {
    if (!this.enableHeartbeat) {
      return; // Skip heartbeat in tests
    }
    
    // Clear existing heartbeat
    if (this.heartbeatInterval !== null) {
      clearInterval(this.heartbeatInterval);
    }

    this.lastHeartbeat = Date.now();

    // Simple heartbeat to detect stale connections
    // We don't send anything, just track when we receive messages
    this.heartbeatInterval = window.setInterval(() => {
      const timeSinceLastHeartbeat = Date.now() - this.lastHeartbeat;
      // On mobile, be more aggressive - if no messages in 30 seconds, reconnect
      // (Backend sends updates frequently, so this should be safe)
      if (timeSinceLastHeartbeat > 30000 && this.connectionState === ConnectionState.CONNECTED) {
        console.log('Connection appears stale (no messages for', timeSinceLastHeartbeat, 'ms), reconnecting...');
        this.reconnect();
      }
    }, 5000); // Check every 5 seconds (more frequent for mobile)
  }

  private reconnect() {
    console.log('Forcing reconnection...');
    
    // Clean up old connection completely
    if (this.ws) {
      this.ws.onopen = null;
      this.ws.onclose = null;
      this.ws.onerror = null;
      this.ws.onmessage = null;
      
      try {
        if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
          this.ws.close();
        }
      } catch (e) {
        console.error('Error closing old WebSocket:', e);
      }
      this.ws = null;
    }
    
    // Reset reconnection state
    this.reconnectAttempts = 0; // Reset attempts for immediate reconnection
    if (this.reconnectTimeout !== null) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    
    // Set state to connecting and attempt connection
    this.setConnectionState(ConnectionState.CONNECTING);
    this.connect();
  }

  private connect() {
    // Clean up existing connection
    if (this.ws) {
      // Remove all event listeners to prevent callbacks on old connection
      this.ws.onopen = null;
      this.ws.onclose = null;
      this.ws.onerror = null;
      this.ws.onmessage = null;
      
      // Close if still open
      if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
        try {
          this.ws.close();
        } catch (e) {
          // Ignore errors during cleanup
        }
      }
      this.ws = null;
    }

    // Clear any pending reconnect timeout
    if (this.reconnectTimeout !== null) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    try {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
        this.setConnectionState(ConnectionState.CONNECTED);
        this.startHeartbeat();
        this.flushMessageQueue();
      };

      this.ws.onclose = (event) => {
        console.log('WebSocket closed:', 
          'code:', event.code, 
          'reason:', event.reason || '(none)',
          'wasClean:', event.wasClean,
          'readyState:', this.ws?.readyState
        );
        
        if (this.manualClose) {
          console.log('Manual close detected, not reconnecting');
          this.setConnectionState(ConnectionState.DISCONNECTED);
          return;
        }

        // Connection was lost unexpectedly
        console.log('Unexpected disconnect, will attempt to reconnect');
        this.setConnectionState(ConnectionState.RECONNECTING);
        this.scheduleReconnect();
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        // onclose will be called after onerror, which will handle reconnection
      };

      this.ws.onmessage = (event: MessageEvent) => {
        // Update heartbeat timestamp on any message
        this.lastHeartbeat = Date.now();
        
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
    } catch (error) {
      console.error('Failed to create WebSocket:', error);
      this.setConnectionState(ConnectionState.RECONNECTING);
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    if (this.manualClose) {
      return;
    }

    if (this.reconnectAttempts >= this.options.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      this.setConnectionState(ConnectionState.DISCONNECTED);
      return;
    }

    // Clear any existing timeout
    if (this.reconnectTimeout !== null) {
      clearTimeout(this.reconnectTimeout);
    }

    this.reconnectAttempts++;
    
    // Exponential backoff with max delay of 30 seconds
    const delay = Math.min(
      this.options.reconnectDelay * Math.pow(1.5, this.reconnectAttempts - 1),
      30000
    );
    
    console.log(`Scheduling reconnect attempt ${this.reconnectAttempts} in ${delay}ms`);
    
    this.reconnectTimeout = window.setTimeout(() => {
      this.reconnectTimeout = null;
      if (!this.manualClose) {
        console.log(`Reconnecting (attempt ${this.reconnectAttempts})...`);
        this.connect();
      }
    }, delay);
  }

  private flushMessageQueue() {
    while (this.messageQueue.length > 0) {
      const command = this.messageQueue.shift()!;
      this.sendImmediate(command);
    }
  }

  private sendImmediate(command: Command) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(command));
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

  send(command: Command) {
    if (this.connectionState === ConnectionState.CONNECTED) {
      this.sendImmediate(command);
    } else {
      this.messageQueue.push(command);
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
    
    // Clean up event listeners
    this.cleanupEventListeners();
    
    // Clear any pending reconnect timeout
    if (this.reconnectTimeout !== null) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    
    if (this.ws) {
      // Remove handlers to prevent reconnection
      this.ws.onclose = null;
      this.ws.onerror = null;
      this.ws.onmessage = null;
      this.ws.onopen = null;
      
      if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
        try {
          this.ws.close();
        } catch (e) {
          // Ignore errors during close
        }
      }
    }
    this.setConnectionState(ConnectionState.DISCONNECTED);
  }
}


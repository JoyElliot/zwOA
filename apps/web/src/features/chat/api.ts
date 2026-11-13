import { request } from '../../shared/api/client';

export interface Contact {
  id: number;
  name: string;
  username: string;
}

export interface Conversation {
  id: number;
  title: string;
  group: boolean;
  memberCount: number;
  lastText: string;
  lastSender: string;
  updatedAt: number;
  unread: number;
  avatarVersion: string;
}

export interface Message {
  id: number;
  senderId: number;
  sender: string;
  text: string;
  sentAt: number;
}

interface MessagePage {
  messages: Message[];
  hasMore: boolean;
  avatars?: { accountId: number; version: string }[];
}

export function newRequestKey() {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

export const getConversations = (signal: AbortSignal) =>
  request<Conversation[]>('/chat/conversations', { signal });

export const findContacts = (query: string, signal: AbortSignal) =>
  request<Contact[]>(`/chat/contacts?q=${encodeURIComponent(query)}`, { signal });

export const createConversation = (body: { title: string; memberIds: number[]; requestKey: string }) =>
  request<{ id: number }>('/chat/conversations', { method: 'POST', body });

export const getMessages = (id: number, signal: AbortSignal, cursor?: { before: number } | { after: number }) =>
  request<MessagePage>(`/chat/conversations/${id}/messages${!cursor ? '' : 'before' in cursor ? `?before=${cursor.before}` : `?after=${cursor.after}`}`, { signal });

export const sendMessage = (id: number, body: { text: string; requestKey: string }) =>
  request<Message>(`/chat/conversations/${id}/messages`, { method: 'POST', body });

export const markRead = (id: number, messageId: number, signal: AbortSignal) =>
  request(`/chat/conversations/${id}/read`, { method: 'PUT', body: { messageId }, signal });

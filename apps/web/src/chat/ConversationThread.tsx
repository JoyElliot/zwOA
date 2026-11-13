import { ArrowLeftOutlined, MessageOutlined, SendOutlined, TeamOutlined } from '@ant-design/icons';
import { Alert, Avatar, Button, Empty, Input, Spin } from 'antd';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { errorMessage } from '../api/client';
import { getMessages, markRead, newRequestKey, sendMessage, type Conversation, type Message } from './api';

const messageTime = new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit' });
const fullTime = new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' });
const dayLabel = new Intl.DateTimeFormat('zh-CN', { dateStyle: 'long' });

export function ChatAvatar({ title, group, size = 38 }: { title: string; group?: boolean; size?: number }) {
  return <Avatar className="chat-avatar" shape="square" size={size} icon={group ? <TeamOutlined /> : undefined}>
    {!group && Array.from(title.trim())[0]?.toUpperCase()}
  </Avatar>;
}

export function ConversationThread({ conversation, actor, onBack, onActivity }: {
  conversation: Conversation; actor: number; onBack: () => void; onActivity: () => void;
}) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [draft, setDraft] = useState('');
  const [initialized, setInitialized] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [olderBusy, setOlderBusy] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState('');
  const [sendError, setSendError] = useState('');
  const history = useRef<HTMLDivElement>(null);
  const initialScroll = useRef(true);
  const atBottom = useRef(true);
  const historyAnchor = useRef<number | null>(null);
  const cursor = useRef(0);
  const readCursor = useRef(0);
  const pendingRead = useRef<number | null>(null);
  const requests = useRef<AbortController | null>(null);
  const busy = useRef(false);
  const sendBusy = useRef(false);
  const lastSend = useRef<{ text: string; requestKey: string } | null>(null);
  const id = conversation.id;

  const readVisible = useCallback(async (signal: AbortSignal) => {
    if (signal.aborted || document.hidden || !atBottom.current || cursor.current <= readCursor.current || pendingRead.current !== null) return;
    const latest = cursor.current;
    pendingRead.current = latest;
    try {
      await markRead(id, latest, signal);
      readCursor.current = latest;
      onActivity();
    } catch {
      // 下一次可见时重新提交已读位置。
    } finally {
      pendingRead.current = null;
    }
  }, [id, onActivity]);

  const sync = useCallback(async (signal: AbortSignal) => {
    if (busy.current || signal.aborted) return;
    busy.current = true;
    try {
      const previous = cursor.current;
      const page = await getMessages(id, signal, previous > 0 ? { after: previous } : undefined);
      if (signal.aborted) return;
      if (previous === 0) setHasMore(page.hasMore);
      const latest = page.messages.at(-1);
      if (latest) {
        cursor.current = latest.id;
        setMessages((current) => {
          const merged = new Map(current.map((message) => [message.id, message]));
          for (const message of page.messages) merged.set(message.id, message);
          return [...merged.values()].sort((a, b) => a.id - b.id);
        });
      }
      setInitialized(true);
      setError('');
      if (previous === 0 || !page.hasMore) void readVisible(signal);
    } catch (cause) {
      if (!signal.aborted) setError(errorMessage(cause));
    } finally {
      busy.current = false;
    }
  }, [id, readVisible]);

  useEffect(() => {
    const controller = new AbortController();
    requests.current = controller;
    const update = () => {
      if (!document.hidden) void sync(controller.signal);
    };
    update();
    const interval = window.setInterval(update, 2500);
    document.addEventListener('visibilitychange', update);
    window.addEventListener('focus', update);
    return () => {
      controller.abort();
      busy.current = false;
      window.clearInterval(interval);
      document.removeEventListener('visibilitychange', update);
      window.removeEventListener('focus', update);
    };
  }, [sync]);

  useLayoutEffect(() => {
    const panel = history.current;
    if (!panel) return;
    if (historyAnchor.current !== null) {
      panel.scrollTop = panel.scrollHeight - historyAnchor.current;
      historyAnchor.current = null;
    } else if (initialScroll.current || atBottom.current) {
      panel.scrollTop = panel.scrollHeight;
    }
    initialScroll.current = false;
  }, [messages]);

  async function loadOlder() {
    const signal = requests.current?.signal;
    const before = messages[0]?.id;
    if (!signal || !before || olderBusy) return;
    setOlderBusy(true);
    try {
      const page = await getMessages(id, signal, { before });
      if (signal.aborted) return;
      const panel = history.current;
      if (panel) historyAnchor.current = panel.scrollHeight - panel.scrollTop;
      atBottom.current = false;
      setMessages((current) => [...page.messages, ...current]);
      setHasMore(page.hasMore);
      setError('');
    } catch (cause) {
      if (!signal.aborted) setError(errorMessage(cause));
    } finally {
      if (!signal.aborted) setOlderBusy(false);
    }
  }

  async function send() {
    const text = draft.trim();
    const signal = requests.current?.signal;
    if (!text || sendBusy.current || !signal || signal.aborted) return;
    if (lastSend.current?.text !== text) lastSend.current = { text, requestKey: newRequestKey() };
    sendBusy.current = true;
    setSending(true);
    setSendError('');
    try {
      const message = await sendMessage(id, lastSend.current);
      lastSend.current = null;
      if (signal.aborted) return;
      setDraft('');
      atBottom.current = true;
      setMessages((current) => current.some(({ id }) => id === message.id) ? current : [...current, message]);
      await sync(signal);
      onActivity();
    } catch (cause) {
      if (!signal.aborted) setSendError(errorMessage(cause));
    } finally {
      sendBusy.current = false;
      if (!signal.aborted) setSending(false);
    }
  }

  return (
    <section className="chat-thread" aria-label="聊天内容">
      <header className="chat-thread-header">
        <Button className="chat-back" type="text" icon={<ArrowLeftOutlined />} aria-label="返回会话" onClick={onBack} />
        <ChatAvatar title={conversation.title} group={conversation.group} />
        <div className="chat-thread-heading">
          <h2 title={conversation.title}>{conversation.title}</h2>
          <span>{conversation.group ? `${conversation.memberCount} 位成员` : '单聊'}</span>
        </div>
      </header>
      {error && <Alert className="chat-error" type="error" showIcon title={error} action={
        <Button size="small" onClick={() => { if (requests.current) void sync(requests.current.signal); }}>重试</Button>
      } />}
      <div className="chat-history" ref={history} onScroll={(event) => {
        const panel = event.currentTarget;
        atBottom.current = panel.scrollHeight - panel.scrollTop - panel.clientHeight < 40;
        if (requests.current) void readVisible(requests.current.signal);
      }}>
        {hasMore && <div className="chat-history-more"><Button size="small" type="link" loading={olderBusy} onClick={() => void loadOlder()}>加载更早消息</Button></div>}
        {!initialized && !error && <div className="chat-center"><Spin aria-label="加载聊天记录" /></div>}
        {initialized && messages.length === 0 && <div className="chat-center">
          <Empty image={<MessageOutlined />} description="还没有消息，发一条消息开始聊天" />
        </div>}
        <div className="chat-messages" role="log" aria-label="消息记录">
          {messages.map((message, index) => {
            const previous = messages[index - 1];
            const newDay = !previous || dayLabel.format(previous.sentAt) !== dayLabel.format(message.sentAt);
            const repeated = !newDay && previous.senderId === message.senderId && message.sentAt - previous.sentAt < 60000;
            const own = message.senderId === actor;
            return <div key={message.id}>
              {newDay && <div className="chat-day"><time dateTime={new Date(message.sentAt).toISOString()}>{dayLabel.format(message.sentAt)}</time></div>}
              <div className={`chat-message${own ? ' chat-message-own' : ''}${repeated ? ' chat-message-continuation' : ''}`}>
                <div className="chat-message-avatar">{!repeated && <ChatAvatar title={message.sender} size={32} />}</div>
                <div className="chat-message-content">
                  {!repeated && <div className="chat-message-meta"><span>{message.sender}</span> <time title={fullTime.format(message.sentAt)} dateTime={new Date(message.sentAt).toISOString()}>{messageTime.format(message.sentAt)}</time></div>}
                  <div className="chat-bubble">{message.text}</div>
                </div>
              </div>
            </div>;
          })}
        </div>
      </div>
      <div className="chat-composer">
        {sendError && <Alert type="error" showIcon title={sendError} />}
        <div className="chat-compose-box">
          <Input.TextArea variant="borderless" aria-label="输入消息" placeholder="输入消息…" value={draft} maxLength={5000}
            disabled={sending || !initialized} autoSize={{ minRows: 2, maxRows: 5 }} onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && (event.ctrlKey || event.metaKey) && !event.nativeEvent.isComposing) {
                event.preventDefault();
                void send();
              }
            }} />
          <div className="chat-compose-actions">
            <span>Ctrl / ⌘ + Enter 发送</span>
            <Button type="primary" icon={<SendOutlined />} aria-label="发送" disabled={!draft.trim()} loading={sending} onClick={() => void send()}>发送</Button>
          </div>
        </div>
      </div>
    </section>
  );
}

import { MessageOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components/card';
import { Alert, Badge, Button, Empty, Input, Spin } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router';
import { ApiError, errorMessage } from '../../shared/api/client';
import { WorkspaceTitlebar } from '../../shared/ui/DesktopTitlebar';
import { getConversations, type Conversation } from './api';
import { ConversationThread } from './ConversationThread';
import { UserAvatar } from '../../shared/ui/UserAvatar';
import { CreateConversation } from './CreateConversation';
import './chat.css';

const clock = new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit' });
const calendar = new Intl.DateTimeFormat('zh-CN', { month: 'numeric', day: 'numeric' });

export default function ChatPage({ actor }: { actor: number }) {
  const { conversationId } = useParams();
  const navigate = useNavigate();
  const [conversations, setConversations] = useState<Conversation[] | null>(null);
  const [search, setSearch] = useState('');
  const [error, setError] = useState('');
  const [creating, setCreating] = useState(false);
  const request = useRef<AbortController | null>(null);
  const refresh = useCallback(async () => {
    request.current?.abort();
    const controller = new AbortController();
    request.current = controller;
    try {
      const result = await getConversations(controller.signal);
      if (!controller.signal.aborted) {
        setConversations(result);
        setError('');
      }
    } catch (cause) {
      if (!controller.signal.aborted) {
        if (cause instanceof ApiError && cause.status === 403) setConversations([]);
        setError(errorMessage(cause));
      }
    }
  }, []);

  useEffect(() => {
    const update = () => { if (!document.hidden) void refresh(); };
    update();
    const interval = window.setInterval(update, 4000);
    document.addEventListener('visibilitychange', update);
    window.addEventListener('focus', update);
    return () => {
      request.current?.abort();
      window.clearInterval(interval);
      document.removeEventListener('visibilitychange', update);
      window.removeEventListener('focus', update);
    };
  }, [refresh]);

  const selection = conversations?.find(({ id }) => String(id) === conversationId);
  const term = search.trim().toLocaleLowerCase('zh-CN');
  const visible = conversations?.filter((item) => item.title.toLocaleLowerCase('zh-CN').includes(term));
  const now = new Date().toDateString();

  return (
    <div className={`chat-layout${conversationId ? ' chat-show-thread' : ''}`}>
      <WorkspaceTitlebar>
        <div className="chat-list-heading">
          <h1>消息</h1>
          <div className="chat-list-actions">
            <Button type="text" icon={<ReloadOutlined />} aria-label="刷新会话" title="刷新会话" onClick={() => void refresh()} />
            <Button type="text" icon={<PlusOutlined />} aria-label="发起会话" title="发起会话" onClick={() => setCreating(true)} />
          </div>
        </div>
      </WorkspaceTitlebar>
      <ProCard className="chat-inbox" styles={{ body: { display: 'flex', flexDirection: 'column', minHeight: 0, padding: 0 } }}>
        <section className="chat-list-panel" aria-label="会话列表">
          <div className="chat-list-header">
            <Input aria-label="搜索会话" placeholder="搜索会话" prefix={<SearchOutlined />} allowClear maxLength={80}
              value={search} onChange={(event) => setSearch(event.target.value)} />
          </div>
          {error && <Alert className="chat-error" type="error" title={error} showIcon />}
          {!visible && !error && <div className="chat-center"><Spin aria-label="加载会话" /></div>}
          {visible?.length === 0 && <Empty className="chat-list-empty" image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={term ? '没有匹配的会话' : '暂无会话'} />}
          <ul className="chat-list">
            {visible?.map((item) => (
              <li key={item.id}>
                <Link className={`chat-row${selection?.id === item.id ? ' chat-row-selected' : ''}`}
                  to={`/chat/${item.id}`} aria-current={selection?.id === item.id ? 'page' : undefined}>
                  <UserAvatar name={item.title} version={item.avatarVersion} group={item.group} className="chat-avatar" size={42} />
                  <div className="chat-row-copy">
                    <div className="chat-row-top">
                      <strong title={item.title}>{item.title}</strong>
                      <time dateTime={new Date(item.updatedAt).toISOString()}>
                        {new Date(item.updatedAt).toDateString() === now ? clock.format(item.updatedAt) : calendar.format(item.updatedAt)}
                      </time>
                    </div>
                    <div className="chat-row-bottom">
                      <span>{item.lastText ? `${item.group ? `${item.lastSender}：` : ''}${item.lastText}` : '暂无消息'}</span>
                      {item.unread > 0 && <Badge count={item.unread} overflowCount={99} title={`${item.unread} 条未读`} />}
                    </div>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      </ProCard>
      {selection ? (
        <ConversationThread key={selection.id} conversation={selection} actor={actor}
          onBack={() => void navigate('/chat')} onActivity={refresh} />
      ) : (
        <section className="chat-thread" aria-label="聊天内容">
          <div className="chat-center">
            {conversationId && !conversations && !error ? <Spin aria-label="正在打开会话" /> : (
              <Empty image={<MessageOutlined />} description={conversationId
                ? error || '找不到此会话'
                : '选择左侧会话开始聊天'}>
                {conversationId && <Link className="back-link" to="/chat">返回会话列表</Link>}
              </Empty>
            )}
          </div>
        </section>
      )}
      {creating && <CreateConversation onClose={() => setCreating(false)} onCreate={async (id) => {
        await refresh();
        setCreating(false);
        void navigate(`/chat/${id}`);
      }} />}
    </div>
  );
}

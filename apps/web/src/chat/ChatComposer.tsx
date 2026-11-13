import { FolderOpenOutlined, SendOutlined, SmileOutlined } from '@ant-design/icons';
import { Alert, Button, Input, Popover, Tooltip } from 'antd';
import type { TextAreaRef } from 'antd/es/input/TextArea';
import { useRef, useState } from 'react';

const maxLength = 5000;
const emoji = ['😀', '😂', '😊', '😉', '😍', '😎', '🤔', '😅', '😢', '😮', '👍', '👏', '🙏', '🎉', '❤️', '👀', '💪', '✅'];

export function ChatComposer({ draft, onDraftChange, onSend, ready, sending, error }: {
  draft: string; onDraftChange: (value: string) => void; onSend: () => void; ready: boolean; sending: boolean; error: string;
}) {
  const textarea = useRef<TextAreaRef>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const editable = ready && !sending;

  function insertEmoji(value: string) {
    const input = textarea.current?.resizableTextArea?.textArea;
    if (!input) return;
    const { selectionStart, selectionEnd } = input;
    const next = draft.slice(0, selectionStart) + value + draft.slice(selectionEnd);
    setPickerOpen(false);
    if (next.length > maxLength) return;
    onDraftChange(next);
    requestAnimationFrame(() => {
      input.focus();
      const caret = selectionStart + value.length;
      input.setSelectionRange(caret, caret);
    });
  }

  return (
    <footer className="chat-composer">
      <div className="chat-composer-inner">
        {error && <Alert type="error" showIcon title={error} />}
        <div className="chat-composer-row">
          <div className="chat-composer-tools" role="group" aria-label="消息工具">
            <Popover trigger="click" placement="topLeft" open={pickerOpen} onOpenChange={setPickerOpen}
              content={<div className="chat-emoji-grid" role="group" aria-label="选择表情">
                {emoji.map((value) => <Button key={value} type="text" aria-label={value} onClick={() => insertEmoji(value)}>{value}</Button>)}
              </div>}>
              <Button type="text" icon={<SmileOutlined />} aria-label="表情" aria-expanded={pickerOpen} disabled={!editable} />
            </Popover>
            <Tooltip title="暂未支持发送文件">
              <span tabIndex={0} aria-label="发送文件，暂未支持">
                <Button type="text" icon={<FolderOpenOutlined />} disabled aria-label="发送文件" />
              </span>
            </Tooltip>
          </div>
          <div className="chat-compose-box">
            <Input.TextArea ref={textarea} variant="borderless" aria-label="输入消息" placeholder="输入消息，Ctrl+Enter 发送"
              value={draft} maxLength={maxLength} disabled={!editable} autoSize={{ minRows: 2, maxRows: 5 }}
              onChange={(event) => onDraftChange(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && (event.ctrlKey || event.metaKey) && !event.nativeEvent.isComposing) {
                  event.preventDefault();
                  setPickerOpen(false);
                  onSend();
                }
              }} />
            <div className="chat-compose-actions">
              <Button type="primary" icon={<SendOutlined />} aria-label="发送" disabled={!ready || !draft.trim()}
                loading={sending} onClick={onSend}>发送</Button>
            </div>
          </div>
        </div>
      </div>
    </footer>
  );
}

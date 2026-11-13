import { PaperClipOutlined, SendOutlined, SmileOutlined } from '@ant-design/icons';
import { Alert, Button, Input, Popover, Tooltip } from 'antd';
import type { TextAreaRef } from 'antd/es/input/TextArea';
import { useRef, useState } from 'react';

const maxLength = 5000;
const draftAutoSize = { minRows: 1, maxRows: 5 };
const emoji = ['😀', '😂', '😊', '😉', '😍', '😎', '🤔', '😅', '😢', '😮', '👍', '👏', '🙏', '🎉', '❤️', '👀', '💪', '✅'];

export function ChatComposer({ draft, onDraftChange, onSend, ready, sending, error }: {
  draft: string; onDraftChange: (value: string) => void; onSend: (input: HTMLTextAreaElement | null) => void;
  ready: boolean; sending: boolean; error: string;
}) {
  const textarea = useRef<TextAreaRef>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const editable = ready && !sending;

  function submit() {
    if (!editable || !draft.trim()) return;
    setPickerOpen(false);
    onSend(textarea.current?.resizableTextArea?.textArea ?? null);
  }

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
        <div className="chat-compose-box">
          <Tooltip title="暂未支持发送文件">
            <span className="chat-compose-attachment" tabIndex={0} aria-label="发送文件，暂未支持">
              <Button className="chat-compose-tool" type="text" shape="circle" icon={<PaperClipOutlined />} disabled aria-label="发送文件" />
            </span>
          </Tooltip>
          <Input.TextArea ref={textarea} className="chat-scrollbar" variant="borderless" aria-label="输入消息"
            placeholder="输入消息" value={draft} maxLength={maxLength}
            disabled={!ready} readOnly={sending} rows={1} autoSize={draft ? draftAutoSize : false}
            onChange={(event) => onDraftChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && (event.ctrlKey || event.metaKey) && !event.nativeEvent.isComposing) {
                event.preventDefault();
                submit();
              }
            }} />
          <Popover trigger="click" placement="topRight" open={pickerOpen} onOpenChange={setPickerOpen}
            content={<div className="chat-emoji-grid" role="group" aria-label="选择表情">
              {emoji.map((value) => <Button key={value} type="text" aria-label={value} onClick={() => insertEmoji(value)}>{value}</Button>)}
            </div>}>
            <Button className="chat-compose-tool" type="text" shape="circle" icon={<SmileOutlined />}
              aria-label="表情" aria-expanded={pickerOpen} disabled={!editable} />
          </Popover>
          <Button className="chat-compose-send" type="primary" shape="circle" icon={<SendOutlined />}
            aria-label="发送" title="发送（Ctrl+Enter）" disabled={!ready || !draft.trim()} loading={sending} onClick={submit} />
        </div>
      </div>
    </footer>
  );
}

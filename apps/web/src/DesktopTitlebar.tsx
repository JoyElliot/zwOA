import { BlockOutlined, BorderOutlined, CloseOutlined, MinusOutlined } from '@ant-design/icons';
import { isTauri } from '@tauri-apps/api/core';
import { getCurrentWindow } from '@tauri-apps/api/window';
import { useEffect, useState } from 'react';

const desktopWindow = isTauri() ? getCurrentWindow() : null;

export function TitlebarDragRegion() {
  return <div className="header-spacer" data-tauri-drag-region={desktopWindow ? '' : undefined} />;
}

export function DesktopWindowControls() {
  const [maximized, setMaximized] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (!desktopWindow) return;
    const appWindow = desktopWindow;
    let disposed = false;
    let unlisten: (() => void) | undefined;

    function updateMaximized() {
      appWindow.isMaximized().then((value) => {
        if (!disposed) setMaximized(value);
      }).catch((error: unknown) => console.error('无法读取窗口状态', error));
    }

    appWindow.onResized(updateMaximized).then((removeListener) => {
      if (disposed) {
        removeListener();
      } else {
        unlisten = removeListener;
        updateMaximized();
      }
    }).catch((error: unknown) => console.error('无法监听窗口尺寸变化', error));

    return () => {
      disposed = true;
      unlisten?.();
    };
  }, []);

  if (!desktopWindow) return null;
  const appWindow = desktopWindow;

  async function runWindowAction(action: () => Promise<void>) {
    try {
      setFailed(false);
      await action();
    } catch (error) {
      console.error('窗口操作失败', error);
      setFailed(true);
    }
  }

  const maximizeLabel = maximized ? '还原窗口' : '最大化窗口';

  return (
    <div className="window-controls" role="group" aria-label="窗口控制">
      {failed && <span className="window-action-error" role="status">窗口操作失败</span>}
      <button type="button" title="最小化窗口" aria-label="最小化窗口"
        onClick={() => void runWindowAction(() => appWindow.minimize())}>
        <MinusOutlined />
      </button>
      <button type="button" title={maximizeLabel} aria-label={maximizeLabel}
        onClick={() => void runWindowAction(() => appWindow.toggleMaximize())}>
        {maximized ? <BlockOutlined /> : <BorderOutlined />}
      </button>
      <button className="window-close" type="button" title="关闭窗口" aria-label="关闭窗口"
        onClick={() => void runWindowAction(() => appWindow.close())}>
        <CloseOutlined />
      </button>
    </div>
  );
}

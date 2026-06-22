import { useState } from 'react';
import { FloatButton, Drawer } from 'antd';
import { RobotOutlined } from '@ant-design/icons';
import ChatPanel from './ChatPanel';

export default function ChatWidget() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <FloatButton
        icon={<RobotOutlined />}
        type="primary"
        onClick={() => setOpen(true)}
        tooltip="AI Assistant"
        style={{ right: 24, bottom: 24 }}
      />
      <Drawer
        title={null}
        placement="right"
        width={420}
        open={open}
        onClose={() => setOpen(false)}
        styles={{ body: { padding: 0 } }}
        closable={false}
      >
        <ChatPanel />
      </Drawer>
    </>
  );
}

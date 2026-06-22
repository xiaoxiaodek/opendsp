import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { UserOutlined, RobotOutlined } from '@ant-design/icons';

interface Message {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export default function MessageList({ messages }: { messages: Message[] }) {
  return (
    <>
      {messages.filter(m => m.role !== 'system').map((msg, i) => (
        <div key={i} style={{
          marginBottom: 12,
          display: 'flex',
          gap: 8,
          flexDirection: msg.role === 'user' ? 'row-reverse' : 'row',
        }}>
          <div style={{
            width: 32, height: 32, borderRadius: '50%',
            background: msg.role === 'user' ? '#1677ff' : '#f0f0f0',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            flexShrink: 0,
          }}>
            {msg.role === 'user'
              ? <UserOutlined style={{ color: '#fff', fontSize: 14 }} />
              : <RobotOutlined style={{ color: '#1677ff', fontSize: 14 }} />
            }
          </div>
          <div style={{
            maxWidth: '80%',
            padding: '8px 12px',
            borderRadius: 12,
            background: msg.role === 'user' ? '#1677ff' : '#f5f5f5',
            color: msg.role === 'user' ? '#fff' : '#333',
            fontSize: 13,
            lineHeight: 1.6,
          }}>
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {msg.content}
            </ReactMarkdown>
          </div>
        </div>
      ))}
    </>
  );
}

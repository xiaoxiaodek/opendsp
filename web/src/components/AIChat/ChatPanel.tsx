import { useState, useRef, useEffect, useCallback } from 'react';
import { Input, Button, Spin, Space, Typography } from 'antd';
import { SendOutlined, RobotOutlined } from '@ant-design/icons';
import { startChat, continueChat, confirmTool } from '../../services/ai';
import MessageList from './MessageList';
import ConfirmDialog from './ConfirmDialog';

const { Text } = Typography;

interface Message {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

interface ConfirmRequest {
  tool_call_id: string;
  tool: string;
  args: Record<string, unknown>;
}

export default function ChatPanel() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [confirmReq, setConfirmReq] = useState<ConfirmRequest | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const decoder = useRef(new TextDecoder());

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const processSSE = useCallback(async (reader: ReadableStreamDefaultReader<Uint8Array>) => {
    let buffer = '';
    let assistantContent = '';

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.current.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          const data = line.slice(6);
          if (data === '[DONE]') {
            if (assistantContent) {
              setMessages(prev => [...prev, { role: 'assistant', content: assistantContent }]);
            }
            setLoading(false);
            return;
          }

          try {
            const parsed = JSON.parse(data);
            if (parsed.session_id) {
              setSessionId(parsed.session_id);
            } else if (parsed.action === 'confirm_required') {
              setConfirmReq({
                tool_call_id: parsed.tool_call_id,
                tool: parsed.tool,
                args: parsed.args || {},
              });
              setLoading(false);
              return;
            } else if (parsed.error) {
              setMessages(prev => [...prev, { role: 'assistant', content: `Error: ${parsed.error}` }]);
              setLoading(false);
              return;
            } else if (typeof parsed === 'string') {
              assistantContent += parsed;
            } else if (parsed.summary || parsed.recommendation) {
              assistantContent += JSON.stringify(parsed, null, 2);
            }
          } catch {
            assistantContent += data;
          }
        }
      }
    } catch {
      setLoading(false);
    }
  }, []);

  const sendMessage = async () => {
    if (!input.trim() || loading) return;
    const msg = input.trim();
    setInput('');
    setMessages(prev => [...prev, { role: 'user', content: msg }]);
    setLoading(true);

    try {
      const reader = sessionId
        ? await continueChat(sessionId, msg)
        : await startChat(msg);
      await processSSE(reader);
    } catch (err) {
      setMessages(prev => [...prev, { role: 'assistant', content: `Error: ${String(err)}` }]);
      setLoading(false);
    }
  };

  const handleConfirm = async (confirmed: boolean) => {
    if (!confirmReq || !sessionId) return;
    setConfirmReq(null);
    setLoading(true);

    try {
      const reader = await confirmTool(sessionId, confirmReq.tool_call_id, confirmed);
      await processSSE(reader);
    } catch (err) {
      setMessages(prev => [...prev, { role: 'assistant', content: `Error: ${String(err)}` }]);
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{
        padding: '12px 16px',
        borderBottom: '1px solid #f0f0f0',
        display: 'flex',
        alignItems: 'center',
        gap: 8,
      }}>
        <RobotOutlined style={{ fontSize: 18, color: '#1677ff' }} />
        <Text strong>AI Assistant</Text>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '12px 16px' }}>
        {messages.length === 0 && (
          <div style={{ color: '#999', textAlign: 'center', marginTop: 40 }}>
            <RobotOutlined style={{ fontSize: 48, color: '#d9d9d9', marginBottom: 16 }} />
            <div>Ask me about your campaigns, reports, or performance.</div>
            <div style={{ marginTop: 8, fontSize: 12 }}>
              Try: "How is my campaign performing today?"
            </div>
          </div>
        )}
        <MessageList messages={messages} />
        {loading && (
          <div style={{ padding: '8px 0' }}>
            <Spin size="small" /> <Text type="secondary" style={{ fontSize: 12 }}>Thinking...</Text>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <div style={{ padding: '12px 16px', borderTop: '1px solid #f0f0f0' }}>
        <Space.Compact style={{ width: '100%' }}>
          <Input
            value={input}
            onChange={e => setInput(e.target.value)}
            onPressEnter={sendMessage}
            placeholder="Ask about your ads..."
            disabled={loading}
          />
          <Button type="primary" icon={<SendOutlined />} onClick={sendMessage} loading={loading} />
        </Space.Compact>
      </div>

      <ConfirmDialog
        open={!!confirmReq}
        tool={confirmReq?.tool || ''}
        args={confirmReq?.args || {}}
        onConfirm={() => handleConfirm(true)}
        onCancel={() => handleConfirm(false)}
      />
    </div>
  );
}

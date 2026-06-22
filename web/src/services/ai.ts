const AI_BASE = '/api/v1/ai';

export interface DashboardInsight {
  summary: string;
  topAdgroup: string;
  worstAdgroup: string;
  pacingAlert: string;
  recommendation: string;
  generatedAt: string;
}

export interface ReportAnomaly {
  hour: string;
  metric: string;
  value: number;
  expected: number;
  explanation: string;
}

export function startChat(message: string): Promise<ReadableStreamDefaultReader<Uint8Array>> {
  const token = localStorage.getItem('token');
  return fetch(`${AI_BASE}/chat`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({ message }),
  }).then(res => {
    if (!res.ok) throw new Error(`Chat error: ${res.status}`);
    return res.body!.getReader();
  });
}

export function continueChat(sessionId: string, message: string): Promise<ReadableStreamDefaultReader<Uint8Array>> {
  const token = localStorage.getItem('token');
  return fetch(`${AI_BASE}/chat/${sessionId}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({ message }),
  }).then(res => {
    if (!res.ok) throw new Error(`Chat error: ${res.status}`);
    return res.body!.getReader();
  });
}

export function confirmTool(sessionId: string, toolCallId: string, confirmed: boolean): Promise<ReadableStreamDefaultReader<Uint8Array>> {
  const token = localStorage.getItem('token');
  return fetch(`${AI_BASE}/chat/${sessionId}/confirm`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({ tool_call_id: toolCallId, confirmed }),
  }).then(res => {
    if (!res.ok) throw new Error(`Confirm error: ${res.status}`);
    return res.body!.getReader();
  });
}

export async function getDashboardInsight(advertiserId?: number): Promise<DashboardInsight> {
  const token = localStorage.getItem('token');
  const params = advertiserId ? `?advertiser_id=${advertiserId}` : '';
  const res = await fetch(`${AI_BASE}/insights/dashboard${params}`, {
    headers: { 'Authorization': `Bearer ${token}` },
  });
  if (!res.ok) throw new Error(`Insight error: ${res.status}`);
  return res.json();
}

export async function getReportAnomalies(startTime: string, endTime: string, advertiserId?: number): Promise<ReportAnomaly[]> {
  const token = localStorage.getItem('token');
  const params = new URLSearchParams({ start_time: startTime, end_time: endTime });
  if (advertiserId) params.set('advertiser_id', String(advertiserId));
  const res = await fetch(`${AI_BASE}/insights/report?${params}`, {
    headers: { 'Authorization': `Bearer ${token}` },
  });
  if (!res.ok) throw new Error(`Anomaly error: ${res.status}`);
  const data = await res.json();
  return data.anomalies || [];
}

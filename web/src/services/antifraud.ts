import { apiGet, apiPost, apiDelete } from './api';

export interface BlacklistEntry {
  id: number;
  rule_type: 'ip' | 'device_id' | 'ua_pattern' | 'geo';
  rule_value: string;
  reason: string;
  created_at: string;
}

export interface FraudEvent {
  id: number;
  request_id: string;
  rule_type: string;
  rule_value: string;
  risk_score: number;
  action: 'blocked' | 'flagged';
  created_at: string;
}

export interface FraudStats {
  total_requests: number;
  blocked: number;
  flagged: number;
  block_rate: number;
}

export async function fetchBlacklist(params: {
  page: number;
  page_size: number;
  rule_type?: string;
}): Promise<{ items: BlacklistEntry[]; total: number }> {
  return apiGet('/api/antifraud/blacklist', params);
}

export async function addBlacklist(entry: Omit<BlacklistEntry, 'id' | 'created_at'>): Promise<BlacklistEntry> {
  return apiPost('/api/antifraud/blacklist', entry);
}

export async function removeBlacklist(id: number): Promise<void> {
  return apiDelete(`/api/antifraud/blacklist/${id}`);
}

export async function fetchFraudEvents(params: {
  page: number;
  page_size: number;
  start_date?: string;
  end_date?: string;
}): Promise<{ items: FraudEvent[]; total: number }> {
  return apiGet('/api/antifraud/events', params);
}

export async function fetchFraudStats(params: {
  start_date?: string;
  end_date?: string;
}): Promise<FraudStats> {
  return apiGet('/api/antifraud/stats', params);
}

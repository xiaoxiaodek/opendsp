import { apiGet } from './api';

export interface Discrepancy {
  date: string;
  advertiser_id: number;
  dsp_cost: number;
  adx_cost: number;
  difference: number;
  difference_pct: number;
}

export interface SettlementSummary {
  total_dsp_cost: number;
  total_adx_cost: number;
  total_discrepancy: number;
  discrepancies: Discrepancy[];
}

export async function fetchSettlement(params: {
  advertiser_id: number;
  start_date: string;
  end_date: string;
}): Promise<SettlementSummary> {
  return apiGet('/api/settlement/reconcile', params);
}

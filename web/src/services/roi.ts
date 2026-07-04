import { apiGet } from './api';

export interface ROIMetrics {
  advertiser_id: number;
  campaign_id?: number;
  adgroup_id?: number;
  date: string;
  cost_micros: number;
  revenue_micros: number;
  conversions: number;
  roas: number;
}

export interface ROISummary {
  total_cost: number;
  total_revenue: number;
  total_conversions: number;
  overall_roas: number;
  daily_metrics: ROIMetrics[];
}

export async function fetchROISummary(params: {
  advertiser_id: number;
  start_date: string;
  end_date: string;
  campaign_id?: number;
}): Promise<ROISummary> {
  return apiGet('/api/roi/summary', params);
}

export async function fetchROIByCampaign(params: {
  advertiser_id: number;
  start_date: string;
  end_date: string;
}): Promise<ROIMetrics[]> {
  return apiGet('/api/roi/by-campaign', params);
}

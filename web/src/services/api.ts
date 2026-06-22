import axios from 'axios';

const api = axios.create({ baseURL: '/api/v1' });

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  res => res,
  err => {
    if (err.response?.status === 401) {
      localStorage.clear();
      window.location.href = '/login';
    }
    return Promise.reject(err);
  },
);

export interface Campaign {
  id: number;
  advertiserId: number;
  name: string;
  budget?: number;
  dailyBudget?: number;
  startTime?: string;
  endTime?: string;
  pacing: number;
  status: number;
  createdAt: string;
  updatedAt: string;
}

export interface AdGroup {
  id: number;
  campaignId: number;
  name: string;
  bidType: number;
  bidPrice: number;
  dailyBudget?: number;
  freqCap?: number;
  targeting: string;
  status: number;
  createdAt: string;
}

export interface Creative {
  id: number;
  adGroupId: number;
  name: string;
  creativeType: number;
  assetUrl: string;
  assetSize?: number;
  assetDuration: number;
  assetWidth: number;
  assetHeight: number;
  assetMime: string;
  title: string;
  description: string;
  ctaText: string;
  brandName: string;
  brandLogo: string;
  landingUrl: string;
  deeplinkUrl: string;
  impTracker: string;
  clickTracker: string;
  auditStatus: number;
  auditReason: string;
  createdAt: string;
}

export interface Dashboard {
  todayCost: number;
  todayImpressions: number;
  todayClicks: number;
  todayCtr: number;
  balance: number;
  activeCampaigns: number;
  activeAdGroups: number;
}

export interface Advertiser {
  id: number;
  name: string;
  industry: string;
  contactName: string;
  contactEmail: string;
  balance: number;
  status: number;
  qualificationStatus: number;
  qualificationReason: string;
  creditLimit: number;
  address: string;
  website: string;
  brandNames: string;
  createdAt: string;
}

export interface ProofMaterial {
  id: number;
  advertiserId: number;
  materialType: number;
  fileUrl: string;
  fileName: string;
  fileSize: number;
  auditStatus: number;
  auditReason: string;
  createdAt: string;
}

export interface BalanceTransaction {
  id: number;
  advertiserId: number;
  amount: number;
  balanceBefore: number;
  balanceAfter: number;
  txType: number;
  description: string;
  createdAt: string;
}

export interface Media {
  id: number;
  name: string;
  code: string;
  domain: string;
}

export interface User {
  id: number;
  email: string;
  name: string;
  advertiserId: number;
  role: string;
  createdAt: string;
}

export interface PendingAudit {
  id: number;
  auditType: number;
  name: string;
  advertiserId: number;
  advertiserName: string;
  status: number;
  reason: string;
  createdAt: string;
}

// Campaign
export const listCampaigns = (advertiserId: number, page = 1, pageSize = 20) =>
  api.get('/campaigns', { params: { advertiser_id: advertiserId, page, page_size: pageSize } });
export const createCampaign = (data: Partial<Campaign>) => api.post('/campaigns', data);
export const updateCampaign = (id: number, data: Partial<Campaign>) => api.patch(`/campaigns/${id}`, data);
export const updateCampaignStatus = (id: number, status: number) => api.patch(`/campaigns/${id}/status`, { status });

// AdGroup
export const listAdGroups = (campaignId: number, page = 1, pageSize = 20) =>
  api.get('/adgroups', { params: { campaign_id: campaignId, page, page_size: pageSize } });
export const createAdGroup = (data: Partial<AdGroup>) => api.post(`/campaigns/${data.campaign_id}/adgroups`, data);
export const updateAdGroupStatus = (id: number, status: number) => api.patch(`/adgroups/${id}/status`, { status });
export const updateAdGroup = (id: number, data: Partial<AdGroup>) => api.patch(`/adgroups/${id}`, data);

// Creative
export const listCreatives = (adGroupId: number, page = 1, pageSize = 20) =>
  api.get('/creatives', { params: { ad_group_id: adGroupId, page, page_size: pageSize } });
export const createCreative = (data: Partial<Creative>) => api.post(`/adgroups/${data.ad_group_id}/creatives`, data);
export const updateCreative = (id: number, data: Partial<Creative>) => api.patch(`/creatives/${id}`, data);

// Dashboard & Report
export const getDashboard = (advertiserId: number) => api.get('/dashboard', { params: { advertiser_id: advertiserId } });
export const getReport = (advertiserId: number, startTime: string, endTime: string) =>
  api.get('/reports', { params: { advertiser_id: advertiserId, start_time: startTime, end_time: endTime } });

export interface EntityReportSubItem {
  id: number;
  name: string;
  impressions: number;
  clicks: number;
  ctr: number;
  cost: number;
  cpm: number;
}

export interface EntityReportHourly {
  hour: string;
  impressions: number;
  clicks: number;
  ctr: number;
  cost: number;
  cpm: number;
}

export interface EntityReport {
  todayCost: number;
  todayImpressions: number;
  todayClicks: number;
  todayCtr: number;
  subItems?: EntityReportSubItem[];
  hourly?: EntityReportHourly[];
}

export const getEntityReport = (advertiserId: number, entityType: string, entityId: number, startTime: string, endTime: string) =>
  api.get('/reports/entity', { params: { advertiser_id: advertiserId, entity_type: entityType, entity_id: entityId, start_time: startTime, end_time: endTime } });

// Advertiser
export const createAdvertiser = (data: Partial<Advertiser>) => api.post('/advertisers', data);
export const listAdvertisers = (page = 1, pageSize = 20, status?: number, qualStatus?: number) =>
  api.get('/advertisers', { params: { page, page_size: pageSize, status, qualification_status: qualStatus } });
export const getAdvertiser = (id: number) => api.get(`/advertisers/${id}`);
export const updateAdvertiser = (id: number, data: Partial<Advertiser>) => api.patch(`/advertisers/${id}`, data);
export const deleteAdvertiser = (id: number) => api.delete(`/advertisers/${id}`);
export const submitQualification = (id: number) => api.post(`/advertisers/${id}/submit`);
export const auditAdvertiser = (id: number, qualification_status: number, reason: string) =>
  api.post(`/advertisers/${id}/audit`, { qualification_status, qualification_reason: reason });

// Proof Material
export const listProofMaterials = (advertiserId: number) => api.get(`/advertisers/${advertiserId}/proofs`);
export const uploadProofMaterial = (advertiserId: number, data: Partial<ProofMaterial>) =>
  api.post(`/advertisers/${advertiserId}/proofs`, data);

// Balance
export const getBalance = (advertiserId: number) => api.get(`/advertisers/${advertiserId}/balance`);
export const recharge = (advertiserId: number, amount: number, description: string) =>
  api.post(`/advertisers/${advertiserId}/recharge`, { amount, description });
export const listTransactions = (advertiserId: number, page = 1, pageSize = 20) =>
  api.get(`/advertisers/${advertiserId}/transactions`, { params: { page, page_size: pageSize } });

// Media
export const createMedia = (data: { name: string; code: string; domain: string }) => api.post('/media', data);
export const updateMedia = (id: number, data: { name?: string; domain?: string }) => api.patch(`/media/${id}`, data);
export const updateMediaStatus = (id: number, status: number) => api.patch(`/media/${id}/status`, { status });

// Admin
export const listUsers = (page = 1, pageSize = 20, role?: string) =>
  api.get('/admin/users', { params: { page, page_size: pageSize, role } });
export const updateUserRole = (id: number, role: string) => api.patch(`/admin/users/${id}/role`, { role });
export const listPendingAudits = (page = 1, pageSize = 20, auditType?: number) =>
  api.get('/admin/audits', { params: { page, page_size: pageSize, audit_type: auditType } });
export const auditCreative = (id: number, auditStatus: number, reason: string) =>
  api.post(`/admin/creatives/${id}/audit`, { audit_status: auditStatus, audit_reason: reason });

// Platform Sync
export interface CreativeSyncStatus {
  creativeId: number;
  platform: string;
  status: number;
  externalId: string;
  externalTvid: string;
  reason: string;
}

export interface AdvertiserSyncStatus {
  advertiserId: number;
  platform: string;
  status: number;
  externalAdId: string;
  reason: string;
}

export const syncCreativeToPlatform = (creativeId: number, platform: string) =>
  api.post(`/sync/creatives/${creativeId}/${platform}`);
export const refreshCreativeSyncStatus = (creativeId: number, platform: string) =>
  api.post(`/sync/creatives/${creativeId}/${platform}/refresh`);
export const syncAdvertiserToPlatform = (advertiserId: number, platform: string) =>
  api.post(`/sync/advertisers/${advertiserId}/${platform}`);

// DMP - Tags
export async function listTags(advertiserId: number, tagType?: number) {
  const { data } = await api.get('/tags', { params: { advertiser_id: advertiserId, tag_type: tagType } })
  return data
}

export async function createTag(params: { name: string; tag_type: number; device_ids?: string[]; device_type?: string }) {
  const { data } = await api.post('/tags', params)
  return data
}

export async function deleteTag(id: number) {
  await api.delete(`/tags/${id}`)
}

// DMP - Audiences
export async function listAudiences(advertiserId: number, audienceType?: number) {
  const { data } = await api.get('/audiences', { params: { advertiser_id: advertiserId, audience_type: audienceType } })
  return data
}

export async function createAudience(params: { name: string; audience_type: number; rules: string }) {
  const { data } = await api.post('/audiences', params)
  return data
}

export async function deleteAudience(id: number) {
  await api.delete(`/audiences/${id}`)
}

// DMP - Lookalike
export async function createLookalike(params: { seed_audience_id: number; name: string; expansion_factor: number }) {
  const { data } = await api.post('/lookalikes', params)
  return data
}

export interface DimensionItem {
  id: number;
  name: string;
  impressions: number;
  clicks: number;
  ctr: number;
  cost: number;
  cpm: number;
}

export const getDashboardBreakdown = (advertiserId: number, dimension: string, topN = 10) =>
  api.get('/dashboard/breakdown', { params: { advertiser_id: advertiserId, dimension, top_n: topN } });

export default api;

export type RiskDispositionStatus = 'open' | 'confirmed' | 'false_positive' | 'ignored' | 'ignore_similar' | 'closed';

export interface RiskDisposition {
  eventId: string;
  status: RiskDispositionStatus;
  note: string;
  scope: 'event' | 'similar' | '';
  fingerprint: string;
  handledBy: string;
  handledAt?: string;
  createdAt: string;
  updatedAt: string;
}

export type RiskDispositionMap = Record<string, RiskDisposition>;

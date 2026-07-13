export type RiskDispositionStatus = 'open' | 'confirmed' | 'ignored';

export interface RiskDisposition {
  eventId: string;
  status: RiskDispositionStatus;
  note: string;
  handledBy: string;
  handledAt?: string;
  createdAt: string;
  updatedAt: string;
}

export type RiskDispositionMap = Record<string, RiskDisposition>;

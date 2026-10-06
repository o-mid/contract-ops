export type CostRow = {
  id: string;
  connectionId: string;
  jobId?: string;
  batchId?: string;
  providerName: string;
  billingAccountId?: string;
  serviceName: string;
  serviceCategory?: string;
  skuId?: string;
  chargeCategory?: string;
  chargePeriodStart: string;
  chargePeriodEnd: string;
  billedCost: string;
  effectiveCost: string;
  billingCurrency: string;
  usageQuantity?: string;
  usageUnit?: string;
  sourceRecordId?: string;
  ingestedAt?: string;
};

export type CostPage = {
  costs: CostRow[];
  total: number;
  nextCursor?: string;
};

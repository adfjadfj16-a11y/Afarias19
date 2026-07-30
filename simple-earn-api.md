# Simple Earn API

## POST /sapi/v1/simple-earn/locked/subscribe

Subscribe to a locked product.

### Request Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| redeemTo | STRING | NO | Destination for redemption proceeds |

---

## GET /sapi/v1/simple-earn/locked/position

Get locked position list.

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| redeemTo | STRING | Destination for redemption proceeds |
| parentPositionId | STRING | Parent position ID |
| rewardAmt | STRING | Reward amount |
| extraRewardAsset | STRING | Extra reward asset |
| extraRewardAPR | STRING | Extra reward APR |
| estExtraRewardAmt | STRING | Estimated extra reward amount |
| nextPay | STRING | Next payment amount |
| nextPayDate | STRING | Next payment date |
| payPeriod | STRING | Payment period |
| redeemAmountEarly | STRING | Redemption amount for early exit |
| rewardsEndDate | STRING | Rewards end date |
| deliverDate | STRING | Delivery date |
| redeemPeriod | STRING | Redemption period |
| redeemingAmt | STRING | Amount being redeemed |
| partialAmtDeliverDate | STRING | Delivery date for partial amount |
| canRedeemEarly | BOOLEAN | Whether early redemption is allowed |
| canFastRedemption | BOOLEAN | Whether fast redemption is allowed |
| autoSubscribe | BOOLEAN | Whether auto-subscribe is enabled |
| type | STRING | Position type |
| status | STRING | Position status |
| canReStake | BOOLEAN | Whether re-staking is allowed |

---

## GET /sapi/v1/simple-earn/flexible/history/subscriptionRecord

Get flexible subscription history.

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| productId | STRING | Product ID |

---

## GET /sapi/v1/simple-earn/locked/history/subscriptionRecord

Get locked subscription history.

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| projectId | STRING | Project ID |

---

## GET /sapi/v1/simple-earn/locked/history/redemptionRecord

Get locked redemption history.

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| originalAmount | STRING | Original redemption amount |
| lossAmount | STRING | Loss amount from early redemption |
| isComplete | BOOLEAN | Whether the redemption is complete |
| rewardAsset | STRING | Reward asset |
| rewardAmt | STRING | Reward amount |
| extraRewardAsset | STRING | Extra reward asset |
| estExtraRewardAmt | STRING | Estimated extra reward amount |

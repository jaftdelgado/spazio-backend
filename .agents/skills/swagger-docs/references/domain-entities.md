# Domain Entities — Spazio Backend

Quick reference for documenting endpoints with accurate business context.

## 1. Properties (core)

| Entity           | Description                                         |
| ---------------- | --------------------------------------------------- |
| `Properties`     | Main listing. Has type, modality, status, and price |
| `PropertyTypes`  | Catalog: residential, commercial, etc.              |
| `Modalities`     | Sale / Rent                                         |
| `PropertyStatus` | Available, Reserved, Sold, etc.                     |

Key fields: `id`, `title`, `description`, `price`, `type_id`, `modality_id`, `status_id`, `location_id`

## 2. Specialization

| Entity                  | Description                                        |
| ----------------------- | -------------------------------------------------- |
| `ResidentialProperties` | Extends Properties: bedrooms, bathrooms, sq meters |
| `CommercialProperties`  | Extends Properties: use type, floor, parking       |
| `Orientations`          | North, South, East, West                           |

## 3. Location & Geography

| Entity      | Description                  |
| ----------- | ---------------------------- |
| `Locations` | Full address of the property |
| `Countries` | Country catalog              |
| `States`    | States / provinces           |
| `Zones`     | Neighborhoods / market zones |

## 4. Financials

| Entity        | Description                                  |
| ------------- | -------------------------------------------- |
| `Prices`      | Price with currency and period (for rentals) |
| `RentPeriods` | Monthly, Annual, etc.                        |

## 5. Services & Clauses

| Entity              | Description                                              |
| ------------------- | -------------------------------------------------------- |
| `Services`          | Included services: electricity, water, internet          |
| `ServiceCategories` | Service groupings                                        |
| `PropertyServices`  | N:M relationship between property and service            |
| `Clauses`           | Contractual conditions                                   |
| `ClauseValueTypes`  | Value type for a clause (text, number)                   |
| `PropertyClauses`   | N:M relationship between property and clause, with value |

## 6. Users & RBAC

| Entity            | Description                         |
| ----------------- | ----------------------------------- |
| `Users`           | System user                         |
| `Roles`           | Agent, Admin, Client, etc.          |
| `Permissions`     | Granular actions                    |
| `RolePermissions` | N:M relationship: role ↔ permission |
| `UserStatus`      | Active, Inactive, Suspended         |

## 7. CRM & Logistics

| Entity           | Description                              |
| ---------------- | ---------------------------------------- |
| `Inquiries`      | Client inquiry / interest in a property  |
| `PropertyAgents` | Agent assignment to a property           |
| `Visits`         | Scheduled visit to a property            |
| `VisitStatus`    | Pending, Confirmed, Completed, Cancelled |

## 8. Operations, Contracts & Payments

| Entity            | Description                          |
| ----------------- | ------------------------------------ |
| `Transactions`    | Purchase or rental operation         |
| `Contracts`       | Contract tied to a transaction       |
| `Payments`        | Payments recorded against a contract |
| `PaymentGateways` | Stripe, Conekta, bank transfer, etc. |
| `PaymentMethods`  | Card, cash, transfer                 |

## 9. Audit & History

All history entities follow the same pattern:

```
{Entity}StatusHistory: id, entity_id, old_status_id, new_status_id, changed_by, changed_at, notes
```

Entities: `PropertyStatusHistory`, `ContractStatusHistory`, `VisitStatusHistory`, `TransactionStatusHistory`

---

## Notes for Documentation

- Catalogs (`PropertyTypes`, `Modalities`, `RentPeriods`, `Zones`, etc.) typically only expose GET endpoints (list and get by ID).
- Specialization entities (`ResidentialProperties`, `CommercialProperties`) are created alongside the property or via a subsequent PATCH.
- History records are **read-only** from the API — they are created internally by triggers or service logic.

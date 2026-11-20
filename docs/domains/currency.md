# Currency domain

Package: [`internal/domain/currency`](../../internal/domain/currency)
Adapter: [`repository/currency_registry.go`](../../internal/platform/database/postgres/repository/currency_registry.go)

## Code and Currency

- `Code` is an ISO 4217 alphabetic code (`NGN`, `USD`, `EUR` are predefined
  constants). `Code.IsValid` checks only the shape (three ASCII uppercase
  letters), not whether the currency is supported. Use the registry for that.
- `Currency` holds `Code`, `Name`, `Exponent` (the number of minor-unit digits,
  e.g. 2 for kobo/cents) and `IsActive`.

The `currencies` table is the source of truth. The seeder adds USD, EUR and NGN,
each with exponent 2 and active.

## Minor units and `FormatMinor`

All amounts are `int64` minor units. To display major units without float
rounding, use:

```go
currency.FormatMinor(-12345, 2) // "-123.45"
currency.FormatMinor(5, 2)      // "0.05"
currency.FormatMinor(100, 0)    // "100"
```

It is exact for every `int64`, including `math.MinInt64`. A fuzz test checks the
round trip: removing the decimal point gives back the original number. The
balance endpoint's `balance_major` uses it.

## Registry

`currency.Registry` is the port services use to check a currency:

```go
type Registry interface {
    Get(code Code) (Currency, bool) // false if unknown OR inactive
    Reload(ctx context.Context) error
}
```

`repository.CurrencyRegistry` implements it as an **in-memory cache**, so the
lookup on every account or transaction create doesn't hit the database:

1. `NewCurrencyRegistry` loads all active currencies at startup. It fails startup
   if that load fails.
2. It starts a goroutine that holds a dedicated pool connection and runs
   `LISTEN currency_updated`.
3. The trigger `trg_currency_updated` calls `pg_notify('currency_updated', code)`
   on any insert, update or delete of `currencies`. The listener then reloads
   the whole cache.
4. If the connection drops, the listener retries every 3 s and re-syncs after
   reconnecting, so changes made during the outage are picked up.
5. The goroutine stops when the server's root context is cancelled.

Reads use a `sync.RWMutex`, and a reload swaps in the new map in one step.
To deactivate a currency, update `is_active` in the database. Every instance
picks up the change within moments, with no restart needed.

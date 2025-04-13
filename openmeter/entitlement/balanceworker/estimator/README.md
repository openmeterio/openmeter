# Balance Worker debounce cache

Given balance worker's output events (snapshots) are only used to trigger notifications we could change the code to only send events when those are required by the downstream services. The solution should be extensible: if we add more servics, we should be able to wire that into the cache.

An entitlement's balance can change due to the following:
- Entitlement Create/Delete
- Entitlement Granting, Grant Voiding
- Entitlement period reset
- Change of underlying feature (we are using by key references)
- Change of underlying meter
- Event ingested

Given the vast majority of recalculations are happening due to event ingestions that is this package's focus. Also given that event ingestion and all the other actions are received on different topics, thus partitions including all in this cache would increase the chance of a race condition.

## Approximating entitlement usage

Even if the ingested events always come on the same partition of a topic (as they are hashed by event's subject), a topic rebalancing or a worker restart can still cause events to be processed partially or fully multiple times.

Topic rebalancing can also cause the cache to move between worker nodes, thus we need a cache that is shared between worker nodes.

Due to these reasons it's extremely hard to have a consistent cache with ClickHouse. Instead we take an approach of estimating the upper bound of usage on an entitlement.

> This means that the "cached" usage of the entitlement is always >= than the actual usage on the entitlement.

When the approximate value hits a treshold, we are just recalculating the entitlement and send the snapshot event (and update the cache with the new value).

### Examples

#### SUM Based meter

We have a sum based meter, the entitlement's value has been already calculated. The value key is `$.value`. We get the following data: `{value: "100"}`

Then we update our approximation of the entitlement's usage by adding 100 to it.

What can go wrong:
- The event is ingested to a previous entitlement period: it's not an issue, as the estimated value will be bigger than the actual value, it would result in earier recalculation
- The event is ingested to a period where the entitlement had an active grant that was later voided: estimated >= usage, we are recalculating maybe in vain
- The event is processed twice (rebalancing of Kafka consumers): see above

Let's say we have the value of `{value: "-10"}`.

In this case the ingested event applied twice might break the estimation >= usage constraint, so we are ignoring negative values. This increases the gap between usage and estimation, but it does not matter as we are recalculating worst case.


##### Infinite estimates

Let's say we have a *parse error* in when calculating the effect of the event. Then we should not stop the processing as it's just an estimation and we can get the data from Entitlements. Instead we add **Infinite** (+inf going forward) to the estimation.

If there are still any thresholds on which we need to recalculate, then `threshold < +inf`, thus we are recalculating. If not, then we just keep the estimate as infinite.


#### Unique count estimates

For unique counts we don't want to maintain the values, so each unique count is estimated to be unique.

## Cache key structure

The cache keys are stored in a form of `entitlement:<ID>:<hash>`, where `<hash>` guarantees that if any of the following happens we get a new unique hash:
- Entitlement Create/Delete
- Entitlement Granting, Grant Voiding
- Entitlement period reset
- Change of underlying feature (we are using by key references)
- Change of underlying meter





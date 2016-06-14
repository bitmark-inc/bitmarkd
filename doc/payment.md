# Some details of transaction payment system

## Definitions:

**tx**
: packed transaction as []byte, including signature

**tx_id**
: SHA3-256(tx)

**asset_id**
: SHA3-512(asset.fingerprint) *however this could change*

**pay_id**
: SHA3-384(concat([tx]))

**proof**
: SHA3-256(concat(pay_id, pay\_nonce, nonce))

**address**
: address in appropriate currency encoding

**fee**
: integral value in currencies lowest unit (quoted to prevent JS weirdness)

**nonce**
: 8…16 byte string represented as quoted hex string

## Fees

Fee applicability is:

1. Assets are bound to an issue and will expire after 600 seconds (10
   min) if no corresponding issue is made.  If an issue is present
   then expiry is tied to issue.  No fee is necessary.  Up to 100
   assets can be sent in one `Assets.Register` RPC.
2. Issues can be made up to a maximum of 100 in one RPC.  All assets
   referenced must have already beed submitted to determine the
   corresponding `asset_id`. Issues will expire after 600 seconds (10
   min) unless a payment is in progress or a proof-of-work nonce is
   provided.
3. Transfers can only be made as a single item. Proof-of-work is **NOT
   allowed**, only a single payment is allowed.

### table of discounts for multiple issues

issue count |    1   |  2→10 | 11→50 | 51→100 | remarks
:-----------|-------:|-------:|-------:|-------:|:------------------------
discount    |     0% |    -5% |    -7% |   -10% |
multiplier  |      1 |    9.5 |   46.5 |     90 | difficulty/fee multiplier

## Issues

After a submitting a `Bitmarks.Issue` RPC the result will be a payment required

```
{
  payment_alternatives: {          // pick one method
    currencyA: [                   // either pay all addresses in currency A
      {address: address1, fee: fee1},
      {address: address2, fee: fee2}
    ],
    currencyB: [                   // or pay all addresses in currency B
      {address: address1, fee: fee1},
      {address: address2, fee: fee2}
    ]
    "*": [                         // or pay all addresses in different currencies
      {currency: currencyA, address: address1, fee: fee1},
      {currency: currencyB, address: address2, fee: fee2}
    ],
  },
  pay_id: hex_identifier,          // to be put into the OP_RETURN of A or B
  pay_nonce: hex_identifier,       // to be as part of proof
  difficulty: hex_256bit           // only applies to issues and is scaled by batch size
}
```

Client must **EITHER:** pay *all*[^all] fee~n~ for *one* of the currencies
**OR:** do a proof of work.

[^all]: Currently, for issue there will only a single accumulated fee
to one address for each currency; therefore the `"*"` will not be
present.


### Payment

Client must embed the `pay_id` into the `OP_RETURN` (or equivalent) into the
currency transaction.  Next the client must send the following
argument data as a `Transaction.Payment` RPC:

```
{
  pay_id: hex_identifier,
  tx_ids: {                        // more entries if the multi-currency("*") option was chosen
    currencyX: {                   // where X is either A or B from above
      tx_id: transaction_id        // from the currency service the tx must have pay_id in its OP_RETURN
  }
}
```

There will be a empty success response if the data can be parsed.

### Proof-of-work

Client will use the bytes from `pay_id` and `pay_nonce` and up to 16
bytes of its own nonce data.  To determine the nonce client keep
iterating this data until the proof calculation meets the required
difficulty.  Next, the client sends the `pay_id` (no need to send
`pay_nonce`) and its computed nonce as argument to a `Transaction.Proof`
RPC:

```
{
  pay_id: hex-identifier           // identifies the block of transactions and corresponding pay_nonce
  nonce: hex-nonce                 // such that proof meets difficulty in request
}
```

The resonse received will contain a verified true/false response
```
{
  verified: bool
}
```

**false ⇒**
: All issues were immediately expired and the client must retry the
  whole issue transaction again, this will only occur if the
  difficulty was not met.

**true  ⇒**
: all issues were immediately verified, the client must wait for
  confirmation before transfer is allowed.

## Transfers

After submitting a `Bitmark.Transfer` RPC the response will contain
the following result:

```
{
  payment_alternatives: {          // pick one method
    currencyA: [                   // either pay all addresses in currency A
      {address: address1, fee: fee1},
      {address: address2, fee: fee2}
    ],
    currencyB: [                   // or pay all addresses in currency B
      {address: address1, fee: fee1},
      {address: address2, fee: fee2}
    ]
    "*": [                         // or pay all addresses in different currencies
      {currency: currencyA, address: address1, fee: fee1},
      {currency: currencyB, address: address2, fee: fee2}
    ],
  },
  pay_id: hex_identifier,          // to be put into the OP_RETURN of A or B
}
```

Client must pay **all** fee~n~[^many] for **one** of the currencies,
embedding the `pay_id` into the `OP_RETURN` (or equivalent) into the
currency transaction.  Next the client must send the following
argument data as a `Transaction.Payment` RPC.:

[^many]: I the default case there will only be one address in each
currency.  In the listed price case there will be two, one will be the
transfer fee and the othe will be the price.

```
{
  option: "payment",               // to select payment
  pay_id: hex_identifier,
  tx_ids: {                        // more entries if the multi-currency("*") option was chosen
    currencyX: {                   // where X is either A or B from above
      tx_id: transaction_id        // from the currency service the tx must have pay_id in its OP_RETURN
  }
}
```

There will be a empty success response if the data can be parsed.


## Summary of operations

1. asset
    * remove duplicates
    * needs a verfied issue to be verified
2. block of issues
    * proof-of-work:
        + compute nonce that meets:  proof < difficulty(batch_size)
        + submit nonce
        + receive `{verified: true/false}`
    * payment
        + no need for nonce computation
        + pay: fee(batch_size) to a recent proofer address (random from recent 100 blocks?)
        + return {{currency, [{address1, fee1}], pay_id}
3. transfer
    * proof-of-work **NOT allowed**
    * payment:
        + pay all fee~n~ to the address~n~ for one currency or multi-currency option.
            a. the default case will only have the fee
            b. the listed amount case will have two items, the fee and the price
        + send the payment transaction ids and payment id

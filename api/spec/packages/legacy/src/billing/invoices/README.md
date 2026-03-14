# Invoice specification structure

This directory contains the invoice specification based on [GOBL](https://docs.gobl.org/). We are fond of the universal nature of GOBL as a great schema for describing complex invoicing problems.

Unfortunately for our use-case we had to change/extend the schema. Firstly OpenMeter operates on draft invoices too, and would want to represent the state as part of the Invoice object. GOBL is intended for invoice generation, we on the other hand should support special line item groups (such as tiered pricing). GOBL's representation is good when considering the invoicing use-case, but until we split the tiered prices into seperate line items we should group them together.

Furthermore it would be somewhat inconvinient to represent time series data in the [cbc.Meta object](https://docs.gobl.org/draft-0/cbc/meta).

GOBL can express way more things that we need for now: by removing the unsupported parts from the schema we can use gobl for PUT/POST request payloads.

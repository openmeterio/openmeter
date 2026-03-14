package meta

// Service layer is not needed in this package (it's just a db wrapper), if this changes, please start adding
// transaction.Run and a proper service layer.
type Service = Adapter

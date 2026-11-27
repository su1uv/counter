package seed

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/su1uv/counter/internal/store"
)

// Write inserts ds in one transaction, parents first, so that on emptied
// tables the identity ids are deterministic. It writes nothing on error.
func Write(ctx context.Context, db *sql.DB, ds Dataset) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insert(ctx, store.New(tx), ds); err != nil {
		return err
	}
	return tx.Commit()
}

func insert(ctx context.Context, q *store.Queries, ds Dataset) error {
	branchIDs := make([]int64, len(ds.Branches))
	for i, b := range ds.Branches {
		id, err := q.InsertBranch(ctx, store.InsertBranchParams{
			Code: b.Code, Name: b.Name, City: b.City, State: b.State,
			Timezone: b.Timezone, OpenedOn: b.OpenedOn, Tier: b.Tier,
		})
		if err != nil {
			return fmt.Errorf("insert branch %s: %w", b.Code, err)
		}
		branchIDs[i] = id
	}

	categoryIDs := make([]int64, len(ds.Categories))
	for i, c := range ds.Categories {
		id, err := q.InsertCategory(ctx, store.InsertCategoryParams{Code: c.Code, Name: c.Name})
		if err != nil {
			return fmt.Errorf("insert category %s: %w", c.Code, err)
		}
		categoryIDs[i] = id
	}

	itemIDs := make([]int64, len(ds.Items))
	for i, it := range ds.Items {
		id, err := q.InsertItem(ctx, store.InsertItemParams{
			Sku: it.SKU, CategoryID: categoryIDs[it.Category], Name: it.Name, Material: it.Material,
			UnitCostCents: it.UnitCostCents, ListPriceCents: it.ListPriceCents, Active: it.Active,
		})
		if err != nil {
			return fmt.Errorf("insert item %s: %w", it.SKU, err)
		}
		itemIDs[i] = id
	}

	employeeIDs := make([]int64, len(ds.Employees))
	for i, e := range ds.Employees {
		id, err := q.InsertEmployee(ctx, store.InsertEmployeeParams{
			Code: e.Code, BranchID: branchIDs[e.Branch], FullName: e.FullName, Role: e.Role, HiredOn: e.HiredOn,
			TerminatedOn: sql.NullTime{Time: e.TerminatedOn, Valid: !e.TerminatedOn.IsZero()},
		})
		if err != nil {
			return fmt.Errorf("insert employee %s: %w", e.Code, err)
		}
		employeeIDs[i] = id
	}

	saleIDs := make([]int64, len(ds.Sales))
	lineIDs := make([][]int64, len(ds.Sales))
	for i, s := range ds.Sales {
		saleID, err := q.InsertSale(ctx, store.InsertSaleParams{
			Folio: s.Folio, BranchID: branchIDs[s.Branch], EmployeeID: employeeIDs[s.Employee], SoldAt: s.SoldAt,
			Status: s.Status, SubtotalCents: s.SubtotalCents, TaxCents: s.TaxCents, TotalCents: s.TotalCents,
			ShiftID: sql.NullInt64{},
		})
		if err != nil {
			return fmt.Errorf("insert sale %s: %w", s.Folio, err)
		}
		saleIDs[i] = saleID
		lineIDs[i] = make([]int64, len(s.Lines))
		for j, l := range s.Lines {
			lineID, err := q.InsertSaleLine(ctx, store.InsertSaleLineParams{
				SaleID: saleID, LineNo: int32(l.LineNo), ItemID: itemIDs[l.Item], Quantity: int32(l.Quantity),
				UnitPriceCents: l.UnitPriceCents, UnitCostCents: l.UnitCostCents,
				DiscountCents: l.DiscountCents, TotalCents: l.TotalCents,
			})
			if err != nil {
				return fmt.Errorf("insert sale %s line %d: %w", s.Folio, l.LineNo, err)
			}
			lineIDs[i][j] = lineID
		}
		for _, p := range s.Payments {
			if _, err := q.InsertSalePayment(ctx, store.InsertSalePaymentParams{
				SaleID: saleID, Method: p.Method, AmountCents: p.AmountCents,
				CardBrand: sql.NullString{String: p.CardBrand, Valid: p.CardBrand != ""},
				Last4:     sql.NullString{String: p.Last4, Valid: p.Last4 != ""},
			}); err != nil {
				return fmt.Errorf("insert sale %s %s payment: %w", s.Folio, p.Method, err)
			}
		}
	}

	returnLineIDs := make([][]int64, len(ds.Returns))
	for i, ret := range ds.Returns {
		returnID, err := q.InsertSaleReturn(ctx, store.InsertSaleReturnParams{
			Folio: ret.Folio, SaleID: saleIDs[ret.Sale], EmployeeID: employeeIDs[ret.Employee], ReturnedAt: ret.ReturnedAt,
			SubtotalCents: ret.SubtotalCents, TaxCents: ret.TaxCents, TotalCents: ret.TotalCents,
		})
		if err != nil {
			return fmt.Errorf("insert return %s: %w", ret.Folio, err)
		}
		returnLineIDs[i] = make([]int64, len(ret.Lines))
		for j, l := range ret.Lines {
			lineID, err := q.InsertSaleReturnLine(ctx, store.InsertSaleReturnLineParams{
				ReturnID: returnID, LineNo: int32(l.LineNo), SaleLineID: lineIDs[ret.Sale][l.SaleLine],
				Quantity: int32(l.Quantity), RefundCents: l.RefundCents,
			})
			if err != nil {
				return fmt.Errorf("insert return %s line %d: %w", ret.Folio, l.LineNo, err)
			}
			returnLineIDs[i][j] = lineID
		}
	}

	supplierIDs := make([]int64, len(ds.Suppliers))
	for i, s := range ds.Suppliers {
		id, err := q.InsertSupplier(ctx, store.InsertSupplierParams{
			Code: s.Code, Name: s.Name, City: s.City, LeadTimeDays: int32(s.LeadTimeDays),
		})
		if err != nil {
			return fmt.Errorf("insert supplier %s: %w", s.Code, err)
		}
		supplierIDs[i] = id
	}

	orderLineIDs := make([][]int64, len(ds.PurchaseOrders))
	for i, po := range ds.PurchaseOrders {
		orderID, err := q.InsertPurchaseOrder(ctx, store.InsertPurchaseOrderParams{
			Folio: po.Folio, SupplierID: supplierIDs[po.Supplier], BranchID: branchIDs[po.Branch], OrderedAt: po.OrderedAt,
			ReceivedAt: sql.NullTime{Time: po.ReceivedAt, Valid: !po.ReceivedAt.IsZero()}, Status: po.Status,
		})
		if err != nil {
			return fmt.Errorf("insert purchase order %s: %w", po.Folio, err)
		}
		orderLineIDs[i] = make([]int64, len(po.Lines))
		for j, l := range po.Lines {
			lineID, err := q.InsertPurchaseOrderLine(ctx, store.InsertPurchaseOrderLineParams{
				PurchaseOrderID: orderID, LineNo: int32(l.LineNo), ItemID: itemIDs[l.Item],
				Quantity: int32(l.Quantity), UnitCostCents: l.UnitCostCents,
			})
			if err != nil {
				return fmt.Errorf("insert purchase order %s line %d: %w", po.Folio, l.LineNo, err)
			}
			orderLineIDs[i][j] = lineID
		}
	}

	for _, m := range ds.Movements {
		p := store.InsertInventoryMovementParams{
			BranchID: branchIDs[m.Branch], ItemID: itemIDs[m.Item], MovedAt: m.MovedAt, Kind: m.Kind,
			Quantity: int32(m.Quantity), UnitCostCents: m.UnitCostCents, Reference: m.Reference,
		}
		switch m.Kind {
		case "sale":
			p.SaleLineID = sql.NullInt64{Int64: lineIDs[m.Doc][m.Line], Valid: true}
		case "return":
			p.SaleReturnLineID = sql.NullInt64{Int64: returnLineIDs[m.Doc][m.Line], Valid: true}
		case "receipt":
			p.PurchaseOrderLineID = sql.NullInt64{Int64: orderLineIDs[m.Doc][m.Line], Valid: true}
		}
		if _, err := q.InsertInventoryMovement(ctx, p); err != nil {
			return fmt.Errorf("insert %s movement %s: %w", m.Kind, m.Reference, err)
		}
	}
	return nil
}

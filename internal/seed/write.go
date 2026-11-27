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
			Timezone: b.Timezone, OpenedOn: b.OpenedOn,
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
		})
		if err != nil {
			return fmt.Errorf("insert employee %s: %w", e.Code, err)
		}
		employeeIDs[i] = id
	}

	for _, s := range ds.Sales {
		saleID, err := q.InsertSale(ctx, store.InsertSaleParams{
			Folio: s.Folio, BranchID: branchIDs[s.Branch], EmployeeID: employeeIDs[s.Employee], SoldAt: s.SoldAt,
			SubtotalCents: s.SubtotalCents, TaxCents: s.TaxCents, TotalCents: s.TotalCents,
		})
		if err != nil {
			return fmt.Errorf("insert sale %s: %w", s.Folio, err)
		}
		for _, l := range s.Lines {
			if _, err := q.InsertSaleLine(ctx, store.InsertSaleLineParams{
				SaleID: saleID, LineNo: int32(l.LineNo), ItemID: itemIDs[l.Item], Quantity: int32(l.Quantity),
				UnitPriceCents: l.UnitPriceCents, UnitCostCents: l.UnitCostCents,
				DiscountCents: l.DiscountCents, TotalCents: l.TotalCents,
			}); err != nil {
				return fmt.Errorf("insert sale %s line %d: %w", s.Folio, l.LineNo, err)
			}
		}
	}
	return nil
}

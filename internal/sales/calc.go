package sales

import (
	"context"
	"fmt"
	"strings"

	"pos/internal/catalog"
	"pos/internal/domain"
)

// resolvedLine is a cart line after server-side price/tax resolution, before
// order-level discount allocation.
type resolvedLine struct {
	ProductID      *int64
	Name           string
	SKU            string
	Barcode        string
	VariantNames   []string
	ModifierNames  []string
	ModifierAmount domain.Money
	Quantity       int64
	UnitPrice      domain.Money // includes variant + modifier adjustments
	CostSnapshot   domain.Money
	TaxRateBps     int64
	TaxInclusive   bool
	TrackStock     bool
	AvailableStock int64
	LineSubtotal   domain.Money // UnitPrice * Quantity
	LineDiscount   domain.Money
	DiscountType   string
	Notes          string

	// filled during finalize()
	NetBase   domain.Money // LineSubtotal - LineDiscount
	OrderCut  domain.Money // this line's share of the order-level discount
	TaxAmount domain.Money
	LineTotal domain.Money
}

func resolveItem(ctx context.Context, cat *catalog.Service, item ItemInput, canOverridePrice, canUseOpenItem bool) (*resolvedLine, error) {
	if item.Quantity <= 0 {
		return nil, domain.NewError("INVALID_INPUT", "Item quantity must be positive", 400)
	}

	line := &resolvedLine{Quantity: item.Quantity, Notes: item.Notes}

	if item.ProductID == nil {
		if !canUseOpenItem {
			return nil, domain.NewError("FORBIDDEN", "You do not have permission to add an open item", 403)
		}
		if strings.TrimSpace(item.OpenItemName) == "" || item.OpenItemPrice == nil {
			return nil, domain.NewError("INVALID_INPUT", "Open item requires a name and price", 400)
		}
		line.Name = item.OpenItemName
		line.UnitPrice = *item.OpenItemPrice
		line.TaxRateBps = item.OpenItemTaxBps
		line.TrackStock = false
	} else {
		p, err := cat.GetProduct(ctx, *item.ProductID)
		if err != nil {
			return nil, err
		}
		if !p.Active {
			return nil, domain.NewError("PRODUCT_INACTIVE", fmt.Sprintf("%s is not available for sale", p.Name), 400)
		}
		line.ProductID = &p.ID
		line.Name = p.Name
		line.SKU = p.SKU
		if len(p.Barcodes) > 0 {
			line.Barcode = p.Barcodes[0]
		}
		line.CostSnapshot = p.Cost
		line.TaxRateBps = p.TaxRateBps
		line.TaxInclusive = p.TaxInclusive
		line.TrackStock = p.TrackStock
		line.AvailableStock = p.Stock

		unit := p.Price
		if item.UnitPriceOverride != nil {
			if !canOverridePrice {
				return nil, domain.NewError("FORBIDDEN", "You do not have permission to override price", 403)
			}
			unit = *item.UnitPriceOverride
		}

		for _, vs := range item.VariantSelections {
			found := false
			for _, g := range p.VariantGroups {
				for _, v := range g.Variants {
					if v.ID == vs.VariantID {
						unit += v.PriceAdjustment
						line.VariantNames = append(line.VariantNames, v.Name)
						found = true
					}
				}
			}
			if !found {
				return nil, domain.NewError("INVALID_INPUT", "Selected variant does not belong to this product", 400)
			}
		}
		for _, ms := range item.ModifierSelections {
			found := false
			for _, g := range p.ModifierGroups {
				for _, o := range g.Options {
					if o.ID == ms.ModifierOptionID {
						unit += o.PriceAdjustment
						line.ModifierAmount += o.PriceAdjustment
						line.ModifierNames = append(line.ModifierNames, o.Name)
						found = true
					}
				}
			}
			if !found {
				return nil, domain.NewError("INVALID_INPUT", "Selected modifier does not belong to this product", 400)
			}
		}
		line.UnitPrice = unit
	}

	line.LineSubtotal = line.UnitPrice.Mul(item.Quantity)

	switch item.DiscountType {
	case "FIXED":
		line.LineDiscount = item.DiscountAmount
		line.DiscountType = "FIXED"
	case "PERCENT":
		line.LineDiscount = line.LineSubtotal.MulPercentBps(item.DiscountPercentBps)
		line.DiscountType = "PERCENT"
	}
	if line.LineDiscount > line.LineSubtotal {
		line.LineDiscount = line.LineSubtotal
	}
	if line.LineDiscount < 0 {
		return nil, domain.NewError("INVALID_INPUT", "Discount cannot be negative", 400)
	}

	line.NetBase = line.LineSubtotal - line.LineDiscount
	return line, nil
}

// totals holds the fully resolved cart: every line finalized with its share
// of the order-level discount and tax.
type totals struct {
	Lines         []*resolvedLine
	Subtotal      domain.Money
	DiscountTotal domain.Money
	TaxTotal      domain.Money
	Total         domain.Money
}

// finalize allocates the order-level discount proportionally across lines by
// each line's net base, then computes tax per line (extracting it for
// tax-inclusive products, adding it for tax-exclusive ones), and sums the
// authoritative sale total directly from the resulting line totals.
//
// Note: mixing tax-inclusive and tax-exclusive products in the same cart is
// supported line-by-line, but the classic receipt identity
// (Subtotal - Discount + Tax == Total) only holds exactly when every line
// uses the same tax mode — the common case of a single catalog-wide setting.
// Total is always correct; Tax is always the true amount charged/embedded.
func finalize(lines []*resolvedLine, orderDiscountAmount domain.Money, orderDiscountType string, orderDiscountPercentBps int64) (*totals, error) {
	var subtotal, lineDiscountSum, netBaseSum domain.Money
	for _, l := range lines {
		subtotal += l.LineSubtotal
		lineDiscountSum += l.LineDiscount
		netBaseSum += l.NetBase
	}

	var orderDiscount domain.Money
	switch orderDiscountType {
	case "FIXED":
		orderDiscount = orderDiscountAmount
	case "PERCENT":
		orderDiscount = netBaseSum.MulPercentBps(orderDiscountPercentBps)
	}
	if orderDiscount > netBaseSum {
		orderDiscount = netBaseSum
	}
	if orderDiscount < 0 {
		return nil, domain.NewError("INVALID_INPUT", "Discount cannot be negative", 400)
	}

	var allocated domain.Money
	for i, l := range lines {
		if netBaseSum == 0 {
			l.OrderCut = 0
		} else if i == len(lines)-1 {
			l.OrderCut = orderDiscount - allocated // remainder to last line avoids rounding drift
		} else {
			l.OrderCut = domain.Money(int64(orderDiscount) * int64(l.NetBase) / int64(netBaseSum))
			allocated += l.OrderCut
		}

		adjustedBase := l.NetBase - l.OrderCut
		if l.TaxInclusive && l.TaxRateBps > 0 {
			// Extract the embedded tax: netOfTax = base * 10000/(10000+rate).
			l.TaxAmount = adjustedBase - domain.Money(int64(adjustedBase)*10000/(10000+l.TaxRateBps))
			l.LineTotal = adjustedBase
		} else {
			l.TaxAmount = adjustedBase.MulPercentBps(l.TaxRateBps)
			l.LineTotal = adjustedBase + l.TaxAmount
		}
	}

	t := &totals{Lines: lines, Subtotal: subtotal, DiscountTotal: lineDiscountSum + orderDiscount}
	for _, l := range lines {
		t.TaxTotal += l.TaxAmount
		t.Total += l.LineTotal
	}
	return t, nil
}

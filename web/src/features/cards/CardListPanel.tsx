import { Button } from '../../components/ui/Button'
import type { CardResponse } from '../../types/card'
import { formatDate, getCardBadgeClass, getCardDisplayNumber, getCardStatusText } from '../../utils/format'

type CardListPanelProps = {
  cards: CardResponse[]
  selectedCardId: string
  revealedCardDetails: CardResponse | null
  onSelect: (card: CardResponse) => void
}

export function CardListPanel({ cards, selectedCardId, revealedCardDetails, onSelect }: CardListPanelProps) {
  return (
    <section className="subPanel">
      <div className="subPanelHeader">
        <h3>Мои карты</h3>
        <span>{cards.length}</span>
      </div>

      {cards.length === 0 && (
        <div className="empty">Список пуст. Нажми “Загрузить карты” или выпусти новую карту.</div>
      )}

      {cards.length > 0 && (
        <div className="cardList">
          {cards.map((card) => {
            const isSelected = selectedCardId === String(card.id)
            const revealedForCard = isSelected && revealedCardDetails?.id === card.id ? revealedCardDetails : null
            return (
            <Button
              key={card.id}
              className={isSelected ? 'bankCardItem active' : 'bankCardItem'}
              type="button"
              onClick={() => onSelect(card)}
              aria-pressed={isSelected}
            >
              <span className="bankCardItemTop">
                <span className="cardLabel">Карта</span>
                <span className={getCardBadgeClass(card)}>{getCardStatusText(card)}</span>
              </span>
              <span className="cardNumber" onMouseDown={(event) => event.stopPropagation()} onClick={(event) => event.stopPropagation()}>{revealedForCard?.number ? revealedForCard.number.replace(/(\d{4})(?=\d)/g, '$1 ').trim() : getCardDisplayNumber(card)}</span>
              <span className="bankCardMetaGrid">
                <span>
                  <small>Card ID</small>
                  <strong>{card.id}</strong>
                </span>
                <span>
                  <small>Account ID</small>
                  <strong>{card.account_id}</strong>
                </span>
                <span>
                  <small>Expiry</small>
                  <strong>{revealedForCard?.expiry || card.expiry || '-'}</strong>
                </span>
                <span>
                  <small>Created</small>
                  <strong>{formatDate(card.created_at)}</strong>
                </span>
                <span>
                  <small>Closed at</small>
                  <strong>{formatDate(card.closed_at)}</strong>
                </span>
              </span>
            </Button>
            )
          })}
        </div>
      )}
    </section>
  )
}

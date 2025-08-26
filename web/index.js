document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('order-form');
    const input = document.getElementById('order-uid-input');
    const loader = document.getElementById('loader');
    const errorMessageContainer = document.getElementById('error-message');
    const detailsContainer = document.getElementById('order-details-container');
    const tbody = document.getElementById('items-body');

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const orderUid = input.value.trim();
        if (!orderUid) return;

        // Сбрасываем состояние
        detailsContainer.classList.add('hidden');
        errorMessageContainer.classList.add('hidden');
        loader.classList.remove('hidden');
        tbody.innerHTML = '';

        try {
            const res = await fetch(`/order/${orderUid}`);
            if (!res.ok) throw new Error(`Ошибка запроса: ${res.status}`);

            const data = await res.json();

            // Заполнение данных
            document.getElementById('order-uid').textContent = data.order_uid;
            document.getElementById('track-number').textContent = data.track_number;
            document.getElementById('customer-id').textContent = data.customer_id;
            document.getElementById('date-created').textContent =
                new Date(data.date_created).toLocaleString();

            document.getElementById('delivery-name').textContent = data.delivery.name;
            document.getElementById('delivery-phone').textContent = data.delivery.phone;
            document.getElementById('delivery-email').textContent = data.delivery.email;
            document.getElementById('delivery-city').textContent = data.delivery.city;
            document.getElementById('delivery-region').textContent = data.delivery.region;
            document.getElementById('delivery-address').textContent = data.delivery.address;
            document.getElementById('delivery-zip').textContent = data.delivery.zip;

            document.getElementById('payment-transaction').textContent = data.payment.transaction;
            document.getElementById('payment-currency').textContent = data.payment.currency;
            document.getElementById('payment-amount').textContent = data.payment.amount;
            document.getElementById('payment-delivery-cost').textContent = data.payment.delivery_cost;
            document.getElementById('payment-goods-total').textContent = data.payment.goods_total;
            document.getElementById('payment-date').textContent =
                new Date(data.payment.payment_dt * 1000).toLocaleString();
            document.getElementById('payment-bank').textContent = data.payment.bank;
            document.getElementById('payment-provider').textContent = data.payment.provider;

            // Товары
            data.items.forEach(item => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
          <td class="border px-4 py-2">${item.chrt_id}</td>
          <td class="border px-4 py-2">${item.name}</td>
          <td class="border px-4 py-2">${item.brand}</td>
          <td class="border px-4 py-2">${item.price}</td>
          <td class="border px-4 py-2">${item.sale}</td>
          <td class="border px-4 py-2">${item.total_price}</td>
        `;
                tbody.appendChild(tr);
            });

            // Показываем блок с деталями
            detailsContainer.classList.remove('hidden');

        } catch (err) {
            console.error(err);
            errorMessageContainer.textContent = err.message;
            errorMessageContainer.classList.remove('hidden');
        } finally {
            loader.classList.add('hidden');
        }
    });
});

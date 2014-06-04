$(function() {
	$('.btn-fk-search').on('click', function() {
		// TODO: Cleaner way to access routes.
		window.open('../../../' + $(this).data('slug') + '/popup/', $(this).data('name'),
			'width=800,toolbar=0,resizable=1,scrollbars=yes,height=600,top=100,left=250');
	});
});

$(function() {
	$('.btn-fk-search').on('click', function() {
		// TODO: Cleaner way to access routes.
		var parts = window.location.pathname.split('/model/')[0];
		window.open(parts + '/model/' + $(this).data('slug') + '/popup/', $(this).data('name'),
			'width=800,toolbar=0,resizable=1,scrollbars=yes,height=600,top=100,left=250');
	});
});

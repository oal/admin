$(function() {
	$('input:first').select();

	$('.btn-fk-search').on('click', function() {
		var prefix = window.location.pathname.split('/')[1];
		window.open('/' + prefix + '/view/' + $(this).data('slug') + '/popup/', $(this).data('name'),
			'width=800,toolbar=0,resizable=1,scrollbars=yes,height=600,top=100,left=250');
	});

	$('a.confirm').on('click', function() {
		var ok = confirm("Are you sure you want to delete this item?");
		if(!ok) {
			return false
		}
	})
});

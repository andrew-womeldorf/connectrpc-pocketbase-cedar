@id("allow-create-book")
permit (
  principal,
  action == Library::Action::"CreateBook",
  resource
);

@id("allow-get-published-book")
permit (
  principal,
  action == Library::Action::"GetBook",
  resource
)
when { resource.status == "published" };

@id("allow-get-own-draft")
permit (
  principal,
  action == Library::Action::"GetBook",
  resource
)
when { resource.status == "draft" && resource.author == principal.id };

@id("allow-list-books")
permit (
  principal,
  action == Library::Action::"ListBooks",
  resource
);

@id("allow-update-own-book")
permit (
  principal,
  action == Library::Action::"UpdateBook",
  resource
)
when { resource.author == principal.id };

@id("allow-delete-own-book")
permit (
  principal,
  action == Library::Action::"DeleteBook",
  resource
)
when { resource.author == principal.id };

@id("allow-create-review")
permit (
  principal,
  action == Library::Action::"CreateReview",
  resource
);

@id("forbid-self-review")
forbid (
  principal,
  action == Library::Action::"CreateReview",
  resource
)
when { resource.book_author == principal.id };

@id("allow-get-review")
permit (
  principal,
  action == Library::Action::"GetReview",
  resource
);

@id("allow-list-reviews")
permit (
  principal,
  action == Library::Action::"ListReviews",
  resource
);

@id("allow-update-own-review")
permit (
  principal,
  action == Library::Action::"UpdateReview",
  resource
)
when { resource.reviewer == principal.id };

@id("allow-delete-own-review")
permit (
  principal,
  action == Library::Action::"DeleteReview",
  resource
)
when { resource.reviewer == principal.id };

@id("allow-moderate-review")
permit (
  principal,
  action == Library::Action::"DeleteReview",
  resource
)
when { resource.book_author == principal.id };

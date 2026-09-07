@id("allow-create-book")
permit (
  principal,
  action == Action::"CreateBook",
  resource
);

@id("allow-get-published-book")
permit (
  principal,
  action == Action::"GetBook",
  resource
)
when { resource.status == "published" };

@id("allow-get-own-draft")
permit (
  principal,
  action == Action::"GetBook",
  resource
)
when { resource.status == "draft" && resource.author == principal.id };

@id("allow-list-books")
permit (
  principal,
  action == Action::"ListBooks",
  resource
);

@id("allow-update-own-book")
permit (
  principal,
  action == Action::"UpdateBook",
  resource
)
when { resource.author == principal.id };

@id("allow-delete-own-book")
permit (
  principal,
  action == Action::"DeleteBook",
  resource
)
when { resource.author == principal.id };

@id("allow-create-review")
permit (
  principal,
  action == Action::"CreateReview",
  resource
);

@id("forbid-self-review")
forbid (
  principal,
  action == Action::"CreateReview",
  resource
)
when { resource.book_author == principal.id };

@id("allow-get-review")
permit (
  principal,
  action == Action::"GetReview",
  resource
);

@id("allow-list-reviews")
permit (
  principal,
  action == Action::"ListReviews",
  resource
);

@id("allow-update-own-review")
permit (
  principal,
  action == Action::"UpdateReview",
  resource
)
when { resource.reviewer == principal.id };

@id("allow-delete-own-review")
permit (
  principal,
  action == Action::"DeleteReview",
  resource
)
when { resource.reviewer == principal.id };

@id("allow-moderate-review")
permit (
  principal,
  action == Action::"DeleteReview",
  resource
)
when { resource.book_author == principal.id };

from rest_framework.pagination import PageNumberPagination


class StandardPagination(PageNumberPagination):
    """统一分页: page/size 参数, 标准化返回"""

    page_query_param = "page"
    page_size_query_param = "size"
    max_page_size = 200

    def get_paginated_response(self, data):
        return super().get_paginated_response({
            "list": data,
            "total": self.page.paginator.count,
            "page": self.page.number,
            "size": self.get_page_size(self.request),
        })